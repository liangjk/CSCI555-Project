package paxos

import "time"

func (px *Paxos) GetDone(args *DoneArgs, reply *DoneReply) {
	px.mu.Lock()
	reply.Done = px.done[px.me]
	reply.Decided = px.decided
	reply.SrvId = px.me
	px.mu.Unlock()
}

func (px *Paxos) ticker() {
	for !px.isdead() {
		nextWakeup := time.Now().Add(tickerIntv * time.Millisecond)

		numServers := len(px.peers)
		replyCh := make(chan DoneReply, numServers-1)
		args := DoneArgs{}
		for i, peer := range px.peers {
			if i != px.me {
				reply := DoneReply{}
				peer := peer
				go func() {
					if !peer.Call("Paxos.GetDone", &args, &reply) {
						reply.SrvId = -1
					}
					replyCh <- reply
				}()
			}
		}
		replyDone := make([]int, numServers)
		replyDecided := -1
		for i := 1; i < numServers; i++ {
			reply := <-replyCh
			if reply.Decided > replyDecided {
				replyDecided = reply.Decided
			}
			if reply.SrvId >= 0 {
				replyDone[reply.SrvId] = reply.Done
			}
		}

		px.mu.Lock()
		done := px.done[px.me]
		for i, rd := range replyDone {
			if rd > px.done[i] {
				px.done[i] = rd
			}
			if px.done[i] < done {
				done = px.done[i]
			}
		}
		if done > px.startIndex {
			if done > px.startIndex+len(px.instances) {
				done = px.startIndex + len(px.instances)
			}
			for i := px.startIndex; i < done; i++ {
				inst := px.getInstanceL(i, false)
				go px.decide(i, inst, nil)
			}
			instances := px.instances[done-px.startIndex:]
			px.instances = make([]*Instance, len(instances))
			copy(px.instances, instances)
			px.startIndex = done
			px.persistL(false)
		}
		if replyDecided > px.decided {
			px.getInstanceL(replyDecided, true)
		}
		px.mu.Unlock()

		time.Sleep(time.Until(nextWakeup))
	}
}
