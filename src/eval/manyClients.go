package main

import (
	"CSCI555Project/session"
	"strconv"
	"sync"
)

func manyClientTest(clt int, cfg *Config, testname string) {
	var ops = 1000 / clt
	sess := make([]*session.Session, clt)
	path := "manyClientParallel/"
	wg := sync.WaitGroup{}
	wg.Add(clt)
	for i := 0; i < clt; i++ {
		id := i
		go func() {
			defer wg.Done()
			sess[id] = cfg.MakeSession(cfg.All())
			sess[id].Create(path + strconv.Itoa(id))
		}()
	}
	wg.Wait()
	wg.Add(clt)
	cfg.begin(testname)
	for i := 0; i < clt; i++ {
		id := i
		go func() {
			defer wg.Done()
			defer sess[id].Close()
			pathid := path + strconv.Itoa(id)
			for j := 0; j < ops; j++ {
				if j%(ops*clt/100) == 0 {
					print(".")
				}
				if sess[id].Acquire(pathid) != session.OK {
					print("?")
				}
				if sess[id].Release(pathid) != session.OK {
					print("?")
				}
			}
		}()
	}
	wg.Wait()
	cfg.end()
	println("*\n" + testname)
}

func manyClientSpeed(run int) {
	var cltSettings = [4]int{5, 10, 20, 40}
	for _, clt := range cltSettings {
		testname := "Round " + strconv.Itoa(run) + "\nmany clients parallel no contend servers: 5, clients:" + strconv.Itoa(clt)
		networksetting := " reliable, no long delay"
		rfcfg := MakeConfig(5, false, false, false, session.Raft)
		manyClientTest(clt, rfcfg, testname+networksetting+" protocol: Raft")
		rfcfg.Cleanup()
		pxcfg := MakeConfig(5, false, false, false, session.Paxos)
		manyClientTest(clt, pxcfg, testname+networksetting+" protocol: Paxos")
		pxcfg.Cleanup()
		networksetting = " unreliable, no long delay"
		rfcfg = MakeConfig(5, true, false, false, session.Raft)
		manyClientTest(clt, rfcfg, testname+networksetting+" protocol: Raft")
		rfcfg.Cleanup()
		pxcfg = MakeConfig(5, true, false, false, session.Paxos)
		manyClientTest(clt, pxcfg, testname+networksetting+" protocol: Paxos")
		pxcfg.Cleanup()
	}
}

func manyClientContendTest(clt int, cfg *Config, testname string) {
	const ops = 1000
	sess := make([]*session.Session, clt)
	path := "manyClientContend/" + testname
	for i := 0; i < clt; i++ {
		sess[i] = cfg.MakeSession(cfg.All())
	}
	sess[0].Create(path)
	doneCh := make(chan bool)
	ch := make(chan bool, ops)
	wg := sync.WaitGroup{}
	wg.Add(clt)
	cfg.begin(testname)
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
						if sess[id].Release(path) != session.OK {
							print("?")
						}
					}
				}
			}
		}()
	}
	for i := 0; i < ops; i++ {
		if i%(ops/100) == 0 {
			print(".")
		}
		<-ch
	}
	cfg.end()
	close(doneCh)
	wg.Wait()
	println("*\n" + testname)
}

func manyClientContend(run int) {
	var cltSettings = [4]int{5, 10, 20, 40}
	for _, clt := range cltSettings {
		testname := "Round " + strconv.Itoa(run) + "\nmany clients contend servers: 5, clients:" + strconv.Itoa(clt)
		networksetting := " reliable, no long delay"
		rfcfg := MakeConfig(5, false, false, false, session.Raft)
		manyClientContendTest(clt, rfcfg, testname+networksetting+" protocol: Raft")
		rfcfg.Cleanup()
		pxcfg := MakeConfig(5, false, false, false, session.Paxos)
		manyClientContendTest(clt, pxcfg, testname+networksetting+" protocol: Paxos")
		pxcfg.Cleanup()
	}
}
