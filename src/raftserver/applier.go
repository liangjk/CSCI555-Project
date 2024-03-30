package raftserver

import (
	"CSCI555Project/raft"
	"CSCI555Project/session"
)

func (srv *RaftServer) applier() {
	for {
		select {
		case msg, ok := <-srv.applyCh:
			if ok {
				srv.applyMsg(&msg)
			} else {
				return
			}
		case <-srv.doneCh:
			return
		}
	}
}

func (srv *RaftServer) applyMsg(msg *raft.ApplyMsg) {
	if msg.CommandValid {
		op, ok := msg.Command.(Op)
		if ok {
			switch op.Code {
			case Create:
				srv.applyCreate(&op)
			case Remove:
				srv.applyRemove(&op)
			case Acquire:
				srv.applyAcquire(&op)
			case Release:
				srv.applyRelease(&op)
			case Extend:
				srv.applyExtend(&op)
			case Revoke:
				srv.applyRevoke(&op)
			default:
				raft.DPrintf("Unknown operation: %v\n", op)
			}
			return
		}
	}
	raft.DPrintf("Unknown apply message: %v\n", *msg)
}

func (srv *RaftServer) updSessL(sessid int32, seq int64) {
	if seq > srv.sessions[sessid] {
		srv.sessions[sessid] = seq
		srv.getWaitL(sessid).Broadcast()
	}
}

func (srv *RaftServer) applyCreate(op *Op) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if op.Seq <= srv.sessions[op.Sessid] {
		return
	}
	if _, ok := srv.files[op.Path]; !ok {
		file := &session.File{Sessid: -1}
		srv.files[op.Path] = file
	}
	srv.updSessL(op.Sessid, op.Seq)
}

func (srv *RaftServer) applyRemove(op *Op) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if op.Seq <= srv.sessions[op.Sessid] {
		return
	}
	if file, ok := srv.files[op.Path]; ok && file.Sessid == -1 {
		delete(srv.files, op.Path)
	}
	srv.updSessL(op.Sessid, op.Seq)
}

func (srv *RaftServer) applyAcquire(op *Op) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if op.Seq <= srv.sessions[op.Sessid] {
		return
	}
	if file, ok := srv.files[op.Path]; ok && file.Sessid == -1 {
		file.Sessid = op.Sessid
		file.Lease = op.Lease
	}
	srv.updSessL(op.Sessid, op.Seq)
}

func (srv *RaftServer) applyRelease(op *Op) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if op.Seq <= srv.sessions[op.Sessid] {
		return
	}
	if file, ok := srv.files[op.Path]; ok && file.Sessid == op.Sessid {
		file.Sessid = -1
	}
	srv.updSessL(op.Sessid, op.Seq)
}

func (srv *RaftServer) applyExtend(op *Op) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if op.Seq <= srv.sessions[op.Sessid] {
		return
	}
	if file, ok := srv.files[op.Path]; ok && file.Sessid == op.Sessid {
		file.Lease = op.Lease
	}
	srv.updSessL(op.Sessid, op.Seq)
}

func (srv *RaftServer) applyRevoke(op *Op) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if file, ok := srv.files[op.Path]; ok && file.Sessid == op.Sessid && file.Lease == op.Lease {
		file.Sessid = -1
	}
}