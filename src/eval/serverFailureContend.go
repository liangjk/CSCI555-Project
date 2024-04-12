package main

import (
	"CSCI555Project/session"
	"sync"
	"time"
)

func failureContendTest(cfg *Config, testname string, failure func(time.Duration, *Config, chan bool)) {
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
	}
	sess[0].Create(path)
	doneCh := make(chan bool)
	ch := make(chan bool, ops)
	rptch := make(chan bool, ops)
	wg := sync.WaitGroup{}
	wg.Add(clt)
	cfg.begin(testname)
	go failure(failperiod, cfg, doneCh)
	go stastic(statperiod, ch, doneCh)
	for i := 0; i < clt; i++ {
		id := i
		go func() {
			for {
				select {
				case <-doneCh:
					sess[id].Close()
					wg.Done()
					return
				default:
					if sess[id].Acquire(path) == session.OK {
						ch <- true
						rptch <- true
						if sess[id].Release(path) != session.OK {
							print("?")
						}
						ch <- true
					}
				}
			}
		}()
	}
	for i := 1; i <= ops; i++ {
		if i%(ops/100) == 0 {
			print(".")
		}
		<-rptch
	}
	close(doneCh)
	cfg.end()
	wg.Wait()
	println("*\n" + testname)
}

func failureContend() {
	testname := "Failure ops contend servers: 5 clients: 20, reliable no long dealy,"

	failuresetting := " no failure (baseline)"
	rfcfg := MakeConfig(5, false, false, false, session.Raft)
	failureContendTest(rfcfg, testname+failuresetting+" protocol: Raft", noFailure)
	rfcfg.Cleanup()
	pxcfg := MakeConfig(5, false, false, false, session.Paxos)
	failureContendTest(pxcfg, testname+failuresetting+" protocol: Paxos", noFailure)
	pxcfg.Cleanup()

	failuresetting = " leader failure"
	rfcfg = MakeConfig(5, false, false, false, session.Raft)
	failureContendTest(rfcfg, testname+failuresetting+" protocol: Raft", periodicalLeaderFail)
	rfcfg.Cleanup()
	pxcfg = MakeConfig(5, false, false, false, session.Paxos)
	failureContendTest(pxcfg, testname+failuresetting+" protocol: Paxos", periodicalLeaderFail)
	pxcfg.Cleanup()

	failuresetting = " random failure"
	rfcfg = MakeConfig(5, false, false, false, session.Raft)
	failureContendTest(rfcfg, testname+failuresetting+" protocol: Raft", randomFail)
	rfcfg.Cleanup()
	pxcfg = MakeConfig(5, false, false, false, session.Paxos)
	failureContendTest(pxcfg, testname+failuresetting+" protocol: Paxos", randomFail)
	pxcfg.Cleanup()
}
