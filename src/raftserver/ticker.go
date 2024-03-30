package raftserver

import "time"

const TickerMs = 1000

func (srv *RaftServer) checkTermL(term int) {
	if term > srv.nowTerm {
		srv.nowTerm = term
		for _, cond := range srv.wait {
			cond.Broadcast()
		}
	}
}

func (srv *RaftServer) ticker() {
	const d = time.Millisecond * TickerMs
	timer := time.NewTimer(d)
	for {
		select {
		case <-srv.doneCh:
			timer.Stop()
			srv.mu.Lock()
			for _, cond := range srv.wait {
				cond.Broadcast()
			}
			srv.mu.Unlock()
			return
		case <-timer.C:
		}
		srv.mu.Lock()
		term, isLeader := srv.rf.GetState()
		srv.checkTermL(term)
		if isLeader {
			srv.checkLeaseL()
		}
		srv.mu.Unlock()
		timer.Reset(d)
	}
}

const GracePeriod = DefaultLease / 2

func (srv *RaftServer) checkLeaseL() {
	now := time.Now().Unix()
	for path, file := range srv.files {
		if file.Sessid == -1 {
			continue
		}
		if file.Lease+GracePeriod > now {
			op := Op{Revoke, path, file.Sessid, -1, file.Lease}
			go srv.rf.Start(op)
		}
	}
}
