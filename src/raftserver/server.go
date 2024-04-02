package raftserver

import (
	"CSCI555Project/labgob"
	"CSCI555Project/labrpc"
	"CSCI555Project/raft"
	"CSCI555Project/session"
	"sync"
	"time"
)

type RaftServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	files map[string]*session.File

	sessions map[int32]int64

	wait    map[int32]*sync.Cond
	nowTerm int

	doneCh chan bool
}

func (srv *RaftServer) getWaitL(sessid int32) *sync.Cond {
	if cond := srv.wait[sessid]; cond != nil {
		return cond
	}
	cond := sync.NewCond(&srv.mu)
	srv.wait[sessid] = cond
	return cond
}

func (srv *RaftServer) operateL(op *session.Op) bool {
	_, term, isLeader := srv.rf.Start(*op)
	if !isLeader {
		return false
	}
	srv.checkTermL(term)
	for {
		srv.getWaitL(op.Sessid).Wait()
		if term != srv.nowTerm {
			return false
		}
		select {
		case <-srv.doneCh:
			return false
		default:
		}
		if op.Seq <= srv.sessions[op.Sessid] {
			return true
		}
	}
}

func (srv *RaftServer) Create(args *session.PathArgs, reply *session.ErrReply) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if args.Seq <= srv.sessions[args.Sessid] {
		reply.Result = session.RepeatedRequest
		return
	}
	op := &session.Op{Code: session.Create, Path: args.Path, Sessid: args.Sessid, Seq: args.Seq}
	if srv.operateL(op) {
		reply.Result = session.OK
	} else {
		reply.Result = session.WrongLeader
	}
}

func (srv *RaftServer) Remove(args *session.PathArgs, reply *session.ErrReply) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if args.Seq <= srv.sessions[args.Sessid] {
		reply.Result = session.RepeatedRequest
		return
	}
	op := &session.Op{Code: session.Remove, Path: args.Path, Sessid: args.Sessid, Seq: args.Seq}
	if srv.operateL(op) {
		if _, ok := srv.files[args.Path]; ok {
			reply.Result = session.LockBusy
		} else {
			reply.Result = session.OK
		}
	} else {
		reply.Result = session.WrongLeader
	}
}

func (srv *RaftServer) Acquire(args *session.PathArgs, reply *session.ErrReply) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if args.Seq <= srv.sessions[args.Sessid] {
		reply.Result = session.RepeatedRequest
		return
	}
	op := &session.Op{Code: session.Acquire, Path: args.Path, Sessid: args.Sessid, Seq: args.Seq, Lease: time.Now().Add(time.Second * session.DefaultLease).Unix()}
	if srv.operateL(op) {
		if file, ok := srv.files[args.Path]; !ok {
			reply.Result = session.LockNotExist
		} else if file.Sessid != args.Sessid {
			reply.Result = session.LockBusy
		} else {
			reply.Result = session.OK
			reply.Lease = file.Lease
		}
	} else {
		reply.Result = session.WrongLeader
	}
}

func (srv *RaftServer) Release(args *session.PathArgs, reply *session.ErrReply) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if args.Seq <= srv.sessions[args.Sessid] {
		reply.Result = session.RepeatedRequest
		return
	}
	op := &session.Op{Code: session.Release, Path: args.Path, Sessid: args.Sessid, Seq: args.Seq}
	if srv.operateL(op) {
		reply.Result = session.OK
	} else {
		reply.Result = session.WrongLeader
	}
}

func (srv *RaftServer) Extend(args *session.PathArgs, reply *session.ErrReply) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if args.Seq <= srv.sessions[args.Sessid] {
		reply.Result = session.RepeatedRequest
		return
	}
	op := &session.Op{Code: session.Extend, Path: args.Path, Sessid: args.Sessid, Seq: args.Seq, Lease: time.Now().Add(time.Second * session.DefaultLease).Unix()}
	if srv.operateL(op) {
		if file, ok := srv.files[args.Path]; !ok {
			reply.Result = session.LockNotExist
		} else if file.Sessid != args.Sessid {
			reply.Result = session.LockNotAcquired
		} else {
			reply.Result = session.OK
			reply.Lease = file.Lease
		}
	} else {
		reply.Result = session.WrongLeader
	}
}

func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister) *RaftServer {
	labgob.Register(session.Op{})

	srv := new(RaftServer)
	srv.me = me
	srv.applyCh = make(chan raft.ApplyMsg)
	srv.rf = raft.Make(servers, me, persister, srv.applyCh)

	srv.files = make(map[string]*session.File)
	srv.sessions = make(map[int32]int64)

	srv.wait = make(map[int32]*sync.Cond)
	srv.nowTerm = 0

	srv.doneCh = make(chan bool)

	go srv.applier()
	go srv.ticker()

	return srv
}

func (srv *RaftServer) Kill() {
	srv.rf.Kill()
	close(srv.doneCh)
}
