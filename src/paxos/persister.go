package paxos

import (
	"sync"
)

type Persister struct {
	mu            sync.Mutex
	paxosstate    int
	instanceState []PersistInstance
	snapshotuntil int
	snapshot      []byte
	enable        bool
}

func MakePersister(enable bool) *Persister {
	return &Persister{enable: enable}
}

func clone[T any](orig []T) []T {
	x := make([]T, len(orig))
	copy(x, orig)
	return x
}

func (ps *Persister) Copy() *Persister {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	np := MakePersister(ps.enable)
	np.paxosstate = ps.paxosstate
	np.instanceState = ps.instanceState
	return np
}

func (ps *Persister) ReadPaxosState() (int, []PersistInstance) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.paxosstate, clone(ps.instanceState)
}

// Save both state and snapshot as a single atomic action,
// to help avoid them getting out of sync.
func (ps *Persister) Save(paxosstate int, instState []PersistInstance) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.paxosstate = paxosstate
	ps.instanceState = clone(instState)
}

func (ps *Persister) Snapshot(until int, snapshot []byte) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.snapshotuntil = until
	ps.snapshot = clone(snapshot)
}

func (ps *Persister) ReadSnapshot() (int, []byte) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.snapshotuntil, clone(ps.snapshot)
}

type PersistInstance struct {
	prepare, accept int
	value           interface{}
	status          Fate
}

func (psi *PersistInstance) read() *Instance {
	inst := new(Instance)
	inst.prepare = psi.prepare
	inst.accept = psi.accept
	inst.value = psi.value
	inst.status = psi.status
	return inst
}

func (px *Paxos) readPersist(ps *Persister) {
	startIndex, instances := ps.ReadPaxosState()
	px.startIndex = startIndex
	px.done[px.me] = startIndex
	px.decided = startIndex - 1
	for i, pinst := range instances {
		inst := pinst.read()
		px.instances = append(px.instances, inst)
		if inst.status == Decided {
			px.decided = startIndex + i
		} else {
			go px.proposer(startIndex+i, nil, inst)
		}
	}
}

func (px *Paxos) persistL(force bool) {
	if px.persister.enable || force {
		pinsts := make([]PersistInstance, len(px.instances))
		for _, inst := range px.instances {
			inst.mu.Lock()
		}
		for i, inst := range px.instances {
			pinsts[i].prepare = inst.prepare
			pinsts[i].accept = inst.accept
			pinsts[i].value = inst.value
			pinsts[i].status = inst.status
		}
		for _, inst := range px.instances {
			inst.mu.Unlock()
		}
		px.persister.Save(px.startIndex, pinsts)
	}
}

func (px *Paxos) persistInstanceL(seq int, inst *Instance) {
	ps := px.persister
	if ps.enable {
		ps.mu.Lock()
		defer ps.mu.Unlock()
		if seq < ps.paxosstate {
			return
		}
		for seq-ps.paxosstate >= len(ps.instanceState) {
			ps.instanceState = append(ps.instanceState, PersistInstance{status: Pending})
		}
		pinst := &ps.instanceState[seq-ps.paxosstate]
		pinst.prepare = inst.prepare
		pinst.accept = inst.accept
		pinst.value = inst.value
		pinst.status = inst.status
	}
}
