package paxos

import (
	"math/rand"
	"time"
)

const (
	shortIntv  = 500
	longIntv   = 2500
	tickerIntv = 2000
)

func waitBeforeRetry(intvms int64) {
	time.Sleep(time.Duration(intvms+rand.Int63()%intvms) * time.Millisecond)
}

func (px *Paxos) proposer(seq int, v interface{}, inst *Instance) {
	if v == nil {
		waitBeforeRetry(longIntv)
	}
	for !px.isdead() {
		inst.mu.Lock()
		if inst.status == Decided {
			inst.mu.Unlock()
			return
		}
		inst.prepare++
		prepare := inst.prepare
		value := inst.value
		accept := inst.accept
		inst.mu.Unlock()

		numServers := len(px.peers)
		prepareReplyCh := make(chan PrepareReply, numServers-1)
		prepareArgs := PrepareArgs{seq, prepare}
		for i, peer := range px.peers {
			if i != px.me {
				reply := PrepareReply{}
				peer := peer
				go func() {
					peer.Call("Paxos.Prepare", &prepareArgs, &reply)
					prepareReplyCh <- reply
				}()
			}
		}
		votes := 1
		prepareOk := false
		badNetwork := false
	PrepareLoop:
		for i := 1; i < numServers; i++ {
			reply := <-prepareReplyCh
			switch reply.Code {
			case Ok:
				votes++
				if reply.AcPr > accept {
					accept = reply.AcPr
					value = reply.Value
				}
				if votes*2 > numServers {
					prepareOk = true
					break PrepareLoop
				}
			case HasDecided:
				go px.decide(seq, inst, reply.Value)
				return
			case LowPrepare:
				inst.mu.Lock()
				if reply.AcPr > inst.prepare {
					inst.prepare = reply.AcPr
				}
				inst.mu.Unlock()
				break PrepareLoop
			case LateRequest:
				DPrintf("Another server has deleted pending instance %v\n", seq)
				break PrepareLoop
			}
			if (i+1-votes)*2 >= numServers {
				badNetwork = true
				break
			}
		}
		if prepareOk {
			if value == nil {
				value = v
			}
			inst.mu.Lock()
			if inst.status == Decided {
				inst.mu.Unlock()
				return
			}
			if inst.prepare != prepare {
				inst.mu.Unlock()
				waitBeforeRetry(longIntv)
				continue
			}
			inst.accept = prepare
			inst.value = value
			px.persistInstanceL(seq, inst)
			inst.mu.Unlock()
		} else if badNetwork {
			waitBeforeRetry(shortIntv)
			continue
		} else {
			waitBeforeRetry(longIntv)
			continue
		}

		acceptReplyCh := make(chan AcceptReply, numServers-1)
		acceptArgs := AcceptArgs{seq, prepare, value}
		for i, peer := range px.peers {
			if i != px.me {
				reply := AcceptReply{}
				peer := peer
				go func() {
					peer.Call("Paxos.Accept", &acceptArgs, &reply)
					acceptReplyCh <- reply
				}()
			}
		}
		votes = 1
		acceptOk := false
		replyPrepare := prepare
		for i := 1; i < numServers; i++ {
			reply := <-acceptReplyCh
			if reply.Reply == Ok {
				votes++
				if votes*2 > numServers {
					acceptOk = true
					break
				}
			} else {
				if reply.Reply > replyPrepare {
					replyPrepare = reply.Reply
				}
				if (i+1-votes)*2 >= numServers {
					break
				}
			}
		}

		if acceptOk {
			decideArgs := DecideArgs{seq, value}
			for i, peer := range px.peers {
				if i != px.me {
					reply := DecideReply{}
					go peer.Call("Paxos.Decide", &decideArgs, &reply)
				}
			}
			go px.decide(seq, inst, value)
			return
		} else if replyPrepare > prepare {
			inst.mu.Lock()
			if replyPrepare > inst.prepare {
				inst.prepare = replyPrepare
			}
			inst.mu.Unlock()
			waitBeforeRetry(longIntv)
		} else {
			waitBeforeRetry(shortIntv)
		}
	}
}
