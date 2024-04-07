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
						srv.applyCreateL(&op)
					case session.Remove:
						srv.applyRemoveL(&op)
					case session.Acquire:
						srv.applyAcquireL(&op)
					case session.Release:
						srv.applyReleaseL(&op)
					case session.Extend:
						srv.applyExtendL(&op)
					case session.Revoke:
						srv.applyRevokeL(&op)
					default:
						paxos.DPrintf("Unknown operation: %v\n", op)
					}
				} else {
					paxos.DPrintf("Unknown decided instance: %v\n", msg)
				}
			}
			// srv.px.Done(srv.applied)
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

func (srv *PaxosServer) applyCreateL(op *session.Op) {
	if op.Seq <= srv.sessions[op.Sessid] {
		return
	}
	if _, ok := srv.files[op.Path]; !ok {
		file := &session.File{Sessid: -1}
		srv.files[op.Path] = file
	}
	srv.updSessL(op.Sessid, op.Seq)
}

func (srv *PaxosServer) applyRemoveL(op *session.Op) {
	if op.Seq <= srv.sessions[op.Sessid] {
		return
	}
	if file, ok := srv.files[op.Path]; ok && file.Sessid == -1 {
		delete(srv.files, op.Path)
	}
	srv.updSessL(op.Sessid, op.Seq)
}

func (srv *PaxosServer) applyAcquireL(op *session.Op) {
	if op.Seq <= srv.sessions[op.Sessid] {
		return
	}
	if file, ok := srv.files[op.Path]; ok {
		if file.Sessid == -1 {
			file.Sessid = op.Sessid
			file.Lease = op.Lease
		} else if file.Sessid == op.Sessid && file.Lease < op.Lease {
			file.Lease = op.Lease
		}
	}
	srv.updSessL(op.Sessid, op.Seq)
}

func (srv *PaxosServer) applyReleaseL(op *session.Op) {
	if op.Seq <= srv.sessions[op.Sessid] {
		return
	}
	if file, ok := srv.files[op.Path]; ok && (file.Sessid == op.Sessid || file.Sessid == op.Sessid^1) {
		file.Sessid = -1
	}
	srv.updSessL(op.Sessid, op.Seq)
}

func (srv *PaxosServer) applyExtendL(op *session.Op) {
	if op.Seq <= srv.sessions[op.Sessid] {
		return
	}
	if file, ok := srv.files[op.Path]; ok && file.Sessid == op.Sessid^1 {
		file.Lease = op.Lease
	}
	srv.updSessL(op.Sessid, op.Seq)
}

func (srv *PaxosServer) applyRevokeL(op *session.Op) {
	if file, ok := srv.files[op.Path]; ok && file.Sessid == op.Sessid && file.Lease == op.Lease {
		file.Sessid = -1
	}
}
