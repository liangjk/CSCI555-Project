package raft

import (
	"bytes"

	"CSCI555Project/labgob"
)

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persistL(force bool) {
	if rf.persister.enable || force {
		w := new(bytes.Buffer)
		e := labgob.NewEncoder(w)
		e.Encode(rf.currentTerm)
		e.Encode(rf.votedFor)
		e.Encode(rf.logs)
		e.Encode(rf.startIndex)
		raftstate := w.Bytes()
		if rf.startIndex > 0 {
			rf.persister.Save(raftstate, rf.snapshot)
		} else {
			rf.persister.Save(raftstate, nil)
		}
	}
}

func (rf *Raft) Persist() {
	rf.mu.Lock()
	rf.persistL(true)
	rf.mu.Unlock()
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) bool {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return false
	}
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm, votedFor int
	var logs []Log
	var startIndex int
	err := d.Decode(&currentTerm)
	if err != nil {
		DPrintf("Read Persist currentTerm error: %v", err)
		return false
	}
	err = d.Decode(&votedFor)
	if err != nil {
		DPrintf("Read Persist votedFor error: %v", err)
		return false
	}
	err = d.Decode(&logs)
	if err != nil {
		DPrintf("Read Persist logs error: %v", err)
		return false
	}
	err = d.Decode(&startIndex)
	if err != nil {
		DPrintf("Read Persist startIndex error: %v", err)
		return false
	}
	rf.currentTerm = currentTerm
	rf.votedFor = votedFor
	rf.logs = logs
	rf.startIndex = startIndex
	if startIndex > 0 {
		rf.snapshot = rf.persister.ReadSnapshot()
		rf.commitIndex = startIndex + 1
		rf.lastApplied = startIndex + 1
	}
	return true
}
