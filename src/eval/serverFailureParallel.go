package main

import (
	"CSCI555Project/session"
	"strconv"
	"sync"
	"time"
)

func failureParallelTest(cfg *Config, testname string, failure func(time.Duration, *Config, chan bool)) {
	const (
		ops        = 1000
		clt        = 20
		statperiod = time.Millisecond * 1000
		failperiod = time.Second * 10
	)
	sess := make([]*session.Session, clt)
	path := "failureParallel/" + testname
	for i := 0; i < clt; i++ {
		sess[i] = cfg.MakeSession(cfg.All())
		sess[i].Create(path + strconv.Itoa(i))
	}
	doneCh := make(chan bool)
	ch := make(chan bool, ops)
	wg := sync.WaitGroup{}
	wg.Add(clt)
	cfg.begin(testname)
	go failure(failperiod, cfg, doneCh)
	go stastic(statperiod, ch, doneCh)
	for i := 0; i < clt; i++ {
		id := i
		go func() {
			defer wg.Done()
			defer sess[id].Close()
			pathid := path + strconv.Itoa(id)
			for j := 1; j <= ops; j++ {
				if j%(ops*clt/100) == 0 {
					print(".")
				}
				if sess[id].Acquire(pathid) != session.OK {
					print("?")
				}
				ch <- true
				if sess[id].Release(pathid) != session.OK {
					print("?")
				}
				ch <- true
			}
		}()
	}
	wg.Wait()
	close(doneCh)
	cfg.end()
	println("*\n" + testname)
}

func failureParallel(run int) {
	testname := "Round " + strconv.Itoa(run) + "\nfailure ops parallel servers: 5 clients: 20, reliable no long dealy,"

	failuresetting := " no failure (baseline)"
	rfcfg := MakeConfig(5, false, false, false, session.Raft)
	failureParallelTest(rfcfg, testname+failuresetting+" protocol: Raft", noFailure)
	rfcfg.Cleanup()
	pxcfg := MakeConfig(5, false, false, false, session.Paxos)
	failureParallelTest(pxcfg, testname+failuresetting+" protocol: Paxos", noFailure)
	pxcfg.Cleanup()

	failuresetting = " leader failure"
	rfcfg = MakeConfig(5, false, false, false, session.Raft)
	failureParallelTest(rfcfg, testname+failuresetting+" protocol: Raft", periodicalLeaderFail)
	rfcfg.Cleanup()
	pxcfg = MakeConfig(5, false, false, false, session.Paxos)
	failureParallelTest(pxcfg, testname+failuresetting+" protocol: Paxos", periodicalLeaderFail)
	pxcfg.Cleanup()

	failuresetting = " nonleader failure"
	rfcfg = MakeConfig(5, false, false, false, session.Raft)
	failureParallelTest(rfcfg, testname+failuresetting+" protocol: Raft", nonLeaderFail)
	rfcfg.Cleanup()
	pxcfg = MakeConfig(5, false, false, false, session.Paxos)
	failureParallelTest(pxcfg, testname+failuresetting+" protocol: Paxos", nonLeaderFail)
	pxcfg.Cleanup()
}
