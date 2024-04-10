package paxosserver

import (
	"CSCI555Project/labrpc"
	"CSCI555Project/paxos"
	"CSCI555Project/session"
	"encoding/gob"
	"sync"
	"time"
)

type PaxosServer struct {
	mu sync.Mutex
	me int
	px *paxos.Paxos

	files map[string]*session.File

	applied int

	sessions map[int32]int64

	doneCh chan bool
}

const (
	MinWaitMs = 10
	MaxWaitMs = 100
	waitStep  = 10
)

func (srv *PaxosServer) operateL(op *session.Op) {
	const threshold = time.Millisecond * MaxWaitMs
	ms := time.Millisecond * MinWaitMs
	step := waitStep
	pxseq := -1
	for {
		if srv.applied > pxseq {
			pxseq = srv.px.Max() + 1
			srv.px.Start(pxseq, *op)
		}
		srv.mu.Unlock()
		timer := time.NewTimer(ms)
		select {
		case <-srv.doneCh:
			timer.Stop()
			srv.mu.Lock()
			return
		case <-timer.C:
		}
		srv.mu.Lock()
		if op.Seq <= srv.sessions[op.Sessid] {
			return
		}
		if ms < threshold {
			if step > 0 {
				step--
			} else {
				ms <<= 1
				step = waitStep
			}
		}
	}
}

func (srv *PaxosServer) Create(args *session.PathArgs, reply *session.ErrReply) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if args.Seq <= srv.sessions[args.Sessid] {
		reply.Result = session.RepeatedRequest
		return
	}
	op := &session.Op{Code: session.Create, Path: args.Path, Sessid: args.Sessid, Seq: args.Seq}
	srv.operateL(op)
	reply.Result = session.OK
}

func (srv *PaxosServer) Remove(args *session.PathArgs, reply *session.ErrReply) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if args.Seq <= srv.sessions[args.Sessid] {
		reply.Result = session.RepeatedRequest
		return
	}
	op := &session.Op{Code: session.Remove, Path: args.Path, Sessid: args.Sessid, Seq: args.Seq}
	srv.operateL(op)
	if _, ok := srv.files[args.Path]; ok {
		reply.Result = session.LockBusy
	} else {
		reply.Result = session.OK
	}
}

func (srv *PaxosServer) Acquire(args *session.PathArgs, reply *session.ErrReply) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if args.Seq <= srv.sessions[args.Sessid] {
		reply.Result = session.RepeatedRequest
		return
	}
	op := &session.Op{Code: session.Acquire, Path: args.Path, Sessid: args.Sessid, Seq: args.Seq, Lease: time.Now().Add(time.Second * session.DefaultLease).Unix()}
	srv.operateL(op)
	if file, ok := srv.files[args.Path]; !ok {
		reply.Result = session.LockNotExist
	} else if file.Sessid != args.Sessid {
		reply.Result = session.LockBusy
	} else {
		reply.Result = session.OK
		reply.Lease = file.Lease
	}
}

func (srv *PaxosServer) Release(args *session.PathArgs, reply *session.ErrReply) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if args.Seq <= srv.sessions[args.Sessid] {
		reply.Result = session.RepeatedRequest
		return
	}
	op := &session.Op{Code: session.Release, Path: args.Path, Sessid: args.Sessid, Seq: args.Seq}
	srv.operateL(op)
	reply.Result = session.OK
}

func (srv *PaxosServer) Extend(args *session.PathArgs, reply *session.ErrReply) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if args.Seq <= srv.sessions[args.Sessid] {
		reply.Result = session.RepeatedRequest
		return
	}
	op := &session.Op{Code: session.Extend, Path: args.Path, Sessid: args.Sessid, Seq: args.Seq, Lease: time.Now().Add(time.Second * session.DefaultLease).Unix()}
	srv.operateL(op)
	if file, ok := srv.files[args.Path]; !ok {
		reply.Result = session.LockNotExist
	} else if file.Sessid != args.Sessid^1 {
		reply.Result = session.LockNotAcquired
	} else {
		reply.Result = session.OK
		reply.Lease = file.Lease
	}
}

func (srv *PaxosServer) Kill() {
	srv.px.Kill()
	close(srv.doneCh)
}

func StartServer(servers []*labrpc.ClientEnd, me int, persister *paxos.Persister) *PaxosServer {
	gob.Register(session.Op{})

	srv := new(PaxosServer)
	srv.me = me
	srv.px = paxos.Make(servers, me, persister)

	srv.files = make(map[string]*session.File)
	srv.applied = 0
	srv.sessions = make(map[int32]int64)

	srv.doneCh = make(chan bool)

	go srv.applier()
	go srv.ticker()

	return srv
}
