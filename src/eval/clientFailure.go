package main

import (
	"CSCI555Project/session"
	"strconv"
	"sync"
)

func clientFailureTest(cfg *Config, testname string) {
	const (
		ops = 100
		clt = 20
	)
	sess := make([]*session.Session, clt)
	path := "clientFailure/mutex"
	for i := 0; i < clt; i++ {
		sess[i] = cfg.MakeSession(cfg.All())
	}
	sess[0].Create(path)
	doneCh := make(chan bool)
	ch := make(chan int, ops)
	wg := sync.WaitGroup{}
	fn := func(id int) {
		defer wg.Done()
		for {
			select {
			case <-doneCh:
				sess[id].Close()
				return
			default:
				if sess[id].Acquire(path) == session.OK {
					ch <- id
					return
				}
			}
		}
	}
	wg.Add(clt)
	cfg.begin(testname)
	for i := 0; i < clt; i++ {
		go fn(i)
	}
	for i := 0; i < ops; i++ {
		id := <-ch
		print(".")
		cfg.DeleteSession(sess[id])
		sess[id].Close()
		sess[id] = cfg.MakeSession(cfg.All())
		wg.Add(1)
		go fn(id)
	}
	cfg.end()
	close(doneCh)
	wg.Wait()
	println("*\n" + testname)
}

func clientFailure(run int) {
	testname := "Round " + strconv.Itoa(run) + " client failure, reliabl, no long delay,"
	rfcfg := MakeConfig(5, false, false, false, session.Raft)
	clientFailureTest(rfcfg, testname+" protocol: Raft")
	rfcfg.Cleanup()
	pxcfg := MakeConfig(5, false, false, false, session.Paxos)
	clientFailureTest(pxcfg, testname+" protocol: Paxos")
	pxcfg.Cleanup()
}
