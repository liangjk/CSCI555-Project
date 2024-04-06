package paxosserver

import (
	"CSCI555Project/paxos"
	"CSCI555Project/session"
	"time"
)

const ApplyWaitMs = 100

func (srv *PaxosServer) applier() {
	for {
		status, msg := srv.px.Status(srv.applied)
		if status == paxos.Decided {
			srv.mu.Lock()
			if msg != nil {
				op, ok := msg.(session.Op)
				if ok {
					switch op.Code {
					case session.Create:
						srv.applyCreate(&op)
					case session.Remove:
						srv.applyRemove(&op)
					case session.Acquire:
						srv.applyAcquire(&op)
					case session.Release:
						srv.applyRelease(&op)
					case session.Extend:
						srv.applyExtend(&op)
					case session.Revoke:
						srv.applyRevoke(&op)
					default:
						paxos.DPrintf("Unknown operation: %v\n", op)
					}
				} else {
					paxos.DPrintf("Unknown decided instance: %v\n", msg)
				}
			}
			srv.px.Done(srv.applied)
			srv.applied++
			srv.mu.Unlock()
		} else {
			timer := time.NewTimer(time.Millisecond * ApplyWaitMs)
			select {
			case <-srv.doneCh:
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}
}

func (srv *PaxosServer) updSessL(sessid int32, seq int64) {
	if seq > srv.sessions[sessid] {
		srv.sessions[sessid] = seq
	}
}

func (srv *PaxosServer) applyCreate(op *session.Op) {
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

func (srv *PaxosServer) applyRemove(op *session.Op) {
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

func (srv *PaxosServer) applyAcquire(op *session.Op) {
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

func (srv *PaxosServer) applyRelease(op *session.Op) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if op.Seq <= srv.sessions[op.Sessid] {
		return
	}
	if file, ok := srv.files[op.Path]; ok && (file.Sessid == op.Sessid || file.Sessid == op.Sessid^1) {
		file.Sessid = -1
	}
	srv.updSessL(op.Sessid, op.Seq)
}

func (srv *PaxosServer) applyExtend(op *session.Op) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if op.Seq <= srv.sessions[op.Sessid] {
		return
	}
	if file, ok := srv.files[op.Path]; ok && file.Sessid == op.Sessid^1 {
		file.Lease = op.Lease
	}
	srv.updSessL(op.Sessid, op.Seq)
}

func (srv *PaxosServer) applyRevoke(op *session.Op) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if file, ok := srv.files[op.Path]; ok && file.Sessid == op.Sessid && file.Lease == op.Lease {
		file.Sessid = -1
	}
}
