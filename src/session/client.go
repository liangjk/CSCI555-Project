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
	mu sync.Mutex

	servers  []*labrpc.ClientEnd
	sessid   int32
	seq      int64
	protocol string
	leader   int

	alive  map[string]int64
	doneCh chan bool
}

var assignedId int32 = -1

func Init(servers []*labrpc.ClientEnd, ptl Protocol) *Session {
	sess := new(Session)
	sess.servers = servers
	sess.sessid = atomic.AddInt32(&assignedId, 1)
	sess.seq = 0
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
				sess.Release(path)
			}
			sess.mu.Unlock()
			return
		case <-timer.C:
		}
		sess.mu.Lock()
		var expired []string
		for path, lease := range sess.alive {
		retryExtend:
			if time.Now().Unix() >= lease-RefreshThreshold {
				reply := sess.RequestL(path, function)
				switch reply.Result {
				case OK:
					sess.alive[path] = reply.Lease
				case RepeatedRequest:
					goto retryExtend
				default:
					expired = append(expired, path)
				}
			}
		}
		for _, path := range expired {
			delete(sess.alive, path)
		}
		sess.mu.Unlock()
		timer.Reset(d)
	}
}

func (sess *Session) Close() {
	close(sess.doneCh)
}

const retryWait = 1000

func (sess *Session) RequestL(path, function string) *ErrReply {
	sess.seq++
	args := PathArgs{path, sess.sessid, sess.seq}
	sCnt := len(sess.servers)
	retry := sCnt
	for {
		reply := ErrReply{}
		ok := sess.servers[sess.leader].Call(function, &args, &reply)
		if ok && reply.Result != WrongLeader {
			return &reply
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
	sess.mu.Lock()
	defer sess.mu.Unlock()
	return sess.RequestL(path, sess.protocol+"Create").Result
}

func (sess *Session) Remove(path string) Err {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	return sess.RequestL(path, sess.protocol+"Remove").Result
}

func (sess *Session) Acquire(path string) Err {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if _, ok := sess.alive[path]; ok {
		return LockReacquire
	}
	reply := sess.RequestL(path, sess.protocol+"Acquire")
	if reply.Result == OK {
		sess.alive[path] = reply.Lease
	}
	return reply.Result
}

func (sess *Session) Release(path string) Err {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if _, ok := sess.alive[path]; !ok {
		return LockNotAcquired
	}
	defer delete(sess.alive, path)
	return sess.RequestL(path, sess.protocol+"Release").Result
}

func (sess *Session) IsHolding(path string) bool {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if lease, ok := sess.alive[path]; ok && time.Now().Unix() <= lease {
		return true
	}
	return false
}
