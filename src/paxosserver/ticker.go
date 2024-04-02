package paxosserver

import (
	"CSCI555Project/session"
	"time"
)

const TickerMs = 1000

func (srv *PaxosServer) ticker() {
	const d = time.Millisecond * TickerMs
	timer := time.NewTimer(d)
	for {
		select {
		case <-srv.doneCh:
			timer.Stop()
			return
		case <-timer.C:
		}
		srv.checkLease()
		timer.Reset(d)
	}
}

func (srv *PaxosServer) checkLease() {
	srv.mu.Lock()
	now := time.Now().Unix()
	for path, file := range srv.files {
		if file.Sessid == -1 {
			continue
		}
		if file.Lease+session.GracePeriod < now {
			go srv.revoke(path)
		}
	}
	srv.mu.Unlock()
}

func (srv *PaxosServer) revoke(path string) {
	srv.mu.Lock()
	file := srv.files[path]
	op := session.Op{Code: session.Revoke, Path: path, Sessid: file.Sessid, Lease: file.Lease}
	threshold := time.Millisecond * MaxWaitMs
	ms := time.Millisecond * MinWaitMs
	pxseq := -1
	for {
		if srv.applied > pxseq {
			pxseq := srv.px.Max() + 1
			srv.px.Start(pxseq, op)
		}
		srv.mu.Unlock()
		timer := time.NewTimer(ms)
		select {
		case <-srv.doneCh:
			timer.Stop()
			return
		case <-timer.C:
		}
		srv.mu.Lock()
		if nfile := srv.files[path]; nfile == file && nfile.Sessid == op.Sessid && nfile.Lease == op.Lease {
			if ms < threshold {
				ms *= 2
			}
		} else {
			srv.mu.Unlock()
			return
		}
	}
}
