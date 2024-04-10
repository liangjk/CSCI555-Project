package paxosserver

import (
	"CSCI555Project/labrpc"
	"CSCI555Project/paxos"
	"sync"
)

type PaxosServers struct {
	leader  int
	servers []*PaxosServer
	saved   []*paxos.Persister
}

func MakePaxosServers(n int) *PaxosServers {
	srvs := &PaxosServers{}
	srvs.servers = make([]*PaxosServer, n)
	srvs.saved = make([]*paxos.Persister, n)
	return srvs
}

func (srvs *PaxosServers) Kill() {
	for _, srv := range srvs.servers {
		if srv != nil {
			srv.Kill()
		}
	}
}

func (srvs *PaxosServers) Shutdown(i int, mu *sync.Mutex) {
	if srvs.saved[i] != nil {
		srvs.saved[i] = srvs.saved[i].Copy()
	}
	srv := srvs.servers[i]
	if srv != nil {
		mu.Unlock()
		srv.Kill()
		mu.Lock()
		srvs.servers[i] = nil
	}
	if srvs.leader == i {
		srvs.leader++
		if srvs.leader >= len(srvs.servers) {
			srvs.leader = 0
		}
	}
}

func (srvs *PaxosServers) Start(ends []*labrpc.ClientEnd, i int, durable bool) {
	if srvs.servers[i] != nil {
		srvs.saved[i] = srvs.saved[i].Copy()
		go srvs.servers[i].Kill()
	} else if srvs.saved[i] == nil {
		srvs.saved[i] = paxos.MakePersister(durable)
	}
	srvs.servers[i] = StartServer(ends, i, srvs.saved[i])
}

func (srvs *PaxosServers) Service(i int) (*labrpc.Service, *labrpc.Service) {
	return labrpc.MakeService(srvs.servers[i]), labrpc.MakeService(srvs.servers[i].px)
}

func (srvs *PaxosServers) Leader() int {
	return srvs.leader
}
