package session

import (
	"CSCI555Project/labrpc"
	"sync"
	"sync/atomic"
	"time"
)

type Protocol bool

const (
	Raft  Protocol = false
	Paxos Protocol = true
)

type Session struct {
	servers  []*labrpc.ClientEnd
	sessid   int32
	seq      [2]int64
	protocol string
	leader   int

	mu    sync.Mutex
	alive map[string]int64

	doneCh chan bool
}

var assignedId int32 = -2

func Init(servers []*labrpc.ClientEnd, ptl Protocol) *Session {
	sess := new(Session)
	sess.servers = servers
	sess.sessid = atomic.AddInt32(&assignedId, 2)
	if ptl == Raft {
		sess.protocol = "RaftServer."
	} else {
		sess.protocol = "PaxosServer."
	}
	sess.leader = 0
	sess.alive = make(map[string]int64)
	sess.doneCh = make(chan bool)
	go sess.KeepAlive()
	return sess
}

func (sess *Session) Close() {
	close(sess.doneCh)
}

const retryWait = 1000

func (sess *Session) Request(rid int32, path, function string) *ErrReply {
	sess.seq[rid]++
	args := PathArgs{path, sess.sessid + rid, sess.seq[rid]}
	sCnt := len(sess.servers)
	retry := sCnt
	for {
		reply := ErrReply{}
		ok := sess.servers[sess.leader].Call(function, &args, &reply)
		if ok {
			if reply.Result == RepeatedRequest {
				return sess.Request(rid, path, function)
			}
			if reply.Result != WrongLeader {
				return &reply
			}
		}
		sess.leader++
		if sess.leader >= sCnt {
			sess.leader = 0
		}
		retry--
		if retry == 0 {
			timer := time.NewTimer(time.Millisecond * retryWait)
			select {
			case <-sess.doneCh:
				timer.Stop()
				return &ErrReply{Result: Terminated}
			case <-timer.C:
			}
			retry = sCnt
		}
	}
}

func (sess *Session) Create(path string) Err {
	return sess.Request(0, path, sess.protocol+"Create").Result
}

func (sess *Session) Remove(path string) Err {
	return sess.Request(0, path, sess.protocol+"Remove").Result
}

func (sess *Session) Acquire(path string) Err {
	sess.mu.Lock()
	if _, ok := sess.alive[path]; ok {
		sess.mu.Unlock()
		return LockReacquire
	}
	sess.mu.Unlock()
	reply := sess.Request(0, path, sess.protocol+"Acquire")
	if reply.Result == OK {
		sess.mu.Lock()
		sess.alive[path] = reply.Lease
		sess.mu.Unlock()
	}
	return reply.Result
}

func (sess *Session) Release(path string) Err {
	sess.mu.Lock()
	if _, ok := sess.alive[path]; !ok {
		sess.mu.Unlock()
		return LockNotAcquired
	}
	delete(sess.alive, path)
	sess.mu.Unlock()
	return sess.Request(0, path, sess.protocol+"Release").Result
}

func (sess *Session) IsHolding(path string) bool {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	for {
		if lease, ok := sess.alive[path]; ok {
			now := time.Now().Unix()
			if now <= lease {
				return true
			} else if now > lease+GracePeriod {
				delete(sess.alive, path)
				return false
			}
		} else {
			return false
		}
		sess.mu.Unlock()
		time.Sleep(time.Millisecond * RefreshRate)
		sess.mu.Lock()
	}
}

const (
	RefreshRate      = 1000
	RefreshThreshold = 2
)

func (sess *Session) KeepAlive() {
	const d = time.Millisecond * RefreshRate
	function := sess.protocol + "Extend"
	timer := time.NewTimer(d)
	for {
		select {
		case <-sess.doneCh:
			timer.Stop()
			sess.mu.Lock()
			for path := range sess.alive {
				sess.Request(1, path, sess.protocol+"Release")
			}
			sess.mu.Unlock()
			return
		case <-timer.C:
		}
		extend := make([]string, 0)
		sess.mu.Lock()
		now := time.Now().Unix()
		for path, lease := range sess.alive {
			if now >= lease-RefreshThreshold {
				extend = append(extend, path)
			}
		}
		sess.mu.Unlock()
		for _, path := range extend {
		RetryExtend:
			reply := sess.Request(1, path, function)
			switch reply.Result {
			case OK:
				sess.mu.Lock()
				if _, ok := sess.alive[path]; ok {
					sess.alive[path] = reply.Lease
				}
				sess.mu.Unlock()
			case RepeatedRequest:
				goto RetryExtend
			default:
				sess.mu.Lock()
				delete(sess.alive, path)
				sess.mu.Unlock()
			}
		}
		timer.Reset(d)
	}
}
