package raftserver

import (
	"CSCI555Project/labrpc"
	"CSCI555Project/raft"
	"sync"
)

type RaftServers struct {
	servers []*RaftServer
	saved   []*raft.Persister
}

func MakeRaftServers(n int) *RaftServers {
	srvs := &RaftServers{}
	srvs.servers = make([]*RaftServer, n)
	srvs.saved = make([]*raft.Persister, n)
	return srvs
}

func (srvs *RaftServers) Kill() {
	for _, srv := range srvs.servers {
		if srv != nil {
			srv.Kill()
		}
	}
}

func (srvs *RaftServers) Shutdown(i int, mu *sync.Mutex) {
	srv := srvs.servers[i]
	if srv != nil {
		mu.Unlock()
		srv.rf.Persist()
		srv.Kill()
		mu.Lock()
	}
	if srvs.saved[i] != nil {
		srvs.saved[i] = srvs.saved[i].Copy()
	}
	srvs.servers[i] = nil
}

func (srvs *RaftServers) Start(ends []*labrpc.ClientEnd, i int, durable bool) {
	if srvs.servers[i] != nil {
		srvs.saved[i] = srvs.saved[i].Copy()
		go srvs.servers[i].Kill()
	} else if srvs.saved[i] == nil {
		srvs.saved[i] = raft.MakePersister(durable)
	}
	srvs.servers[i] = StartServer(ends, i, srvs.saved[i])
}

func (srvs *RaftServers) Service(i int) (*labrpc.Service, *labrpc.Service) {
	return labrpc.MakeService(srvs.servers[i]), labrpc.MakeService(srvs.servers[i].rf)
}

func (srvs *RaftServers) Leader() int {
	for i, srv := range srvs.servers {
		if srv != nil {
			_, isLeader := srv.rf.GetState()
			if isLeader {
				return i
			}
		}
	}
	return -1
}
