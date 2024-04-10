package main

import (
	"CSCI555Project/session"
	"strconv"
)

func oneClientTest(cfg *Config, testname string) {
	const ops = 1000
	sess := cfg.MakeSession(cfg.All())
	path := "oneClientTest/" + testname
	for sess.Create(path) != session.OK {
	}
	cfg.begin(testname)
	for i := 0; i < ops; i++ {
		if i%(ops/100) == 0 {
			print(".")
		}
		if sess.Acquire(path) != session.OK {
			println("?")
		}
		if sess.Release(path) != session.OK {
			println("?")
		}
	}
	cfg.end()
	sess.Close()
	println("*\n" + testname)
}

func oneClientSpeed(run int) {
	var srvSettings = [5]int{3, 5, 7, 9, 11}
	for _, srv := range srvSettings {
		testname := "Round " + strconv.Itoa(run) + "\none client sequential servers: " + strconv.Itoa(srv)
		networksetting := " reliable, no long delay"
		rfcfg := MakeConfig(srv, false, false, false, session.Raft)
		oneClientTest(rfcfg, testname+networksetting+" protocol: Raft")
		rfcfg.Cleanup()
		pxcfg := MakeConfig(srv, false, false, false, session.Paxos)
		oneClientTest(pxcfg, testname+networksetting+" protocol: Paxos")
		pxcfg.Cleanup()
		networksetting = " reliable, long delay"
		rfcfg = MakeConfig(srv, false, true, false, session.Raft)
		oneClientTest(rfcfg, testname+networksetting+" protocol: Raft")
		rfcfg.Cleanup()
		pxcfg = MakeConfig(srv, false, true, false, session.Paxos)
		oneClientTest(pxcfg, testname+networksetting+" protocol: Paxos")
		pxcfg.Cleanup()
	}
}
