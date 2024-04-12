package main

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	statperiod = time.Millisecond * 1000
	failperiod = time.Second * 3
)

func noFailure(cfg *Config, doneCh chan bool) {
	timer := time.NewTimer(failperiod)
	for {
		select {
		case <-doneCh:
			timer.Stop()
			return
		case <-timer.C:
			timer.Reset(failperiod)
		}
	}
}

func periodicalLeaderFail(cfg *Config, doneCh chan bool) {
	timer := time.NewTimer(failperiod * 4)
	id := -1
	for {
		select {
		case <-doneCh:
			timer.Stop()
			return
		case <-timer.C:
			timer.Reset(failperiod * 4)
			if id >= 0 {
				cfg.StartServer(id)
				cfg.Connect(id, cfg.All())
			}
			id = cfg.Leader()
			cfg.ShutdownServer(id)
		}
	}
}

func nonLeaderFail(cfg *Config, doneCh chan bool) {
	ld := cfg.Leader()
	srvs := make([]int, 0)
	for i := 0; i < 5; i++ {
		if i != ld {
			srvs = append(srvs, i)
		}
	}
	timer := time.NewTimer(failperiod)
	updown := false
	id := 0
	for {
		select {
		case <-doneCh:
			timer.Stop()
			return
		case <-timer.C:
			timer.Reset(failperiod)
			if updown {
				cfg.StartServer(srvs[id])
				cfg.Connect(srvs[id], cfg.All())
			} else {
				cfg.ShutdownServer(srvs[id])
			}
			id++
			if id >= 4 {
				id = 0
				updown = !updown
			}
		}
	}
}

func randomFail(cfg *Config, doneCh chan bool) {
	timer := time.NewTimer(failperiod)
	id := -1
	for {
		select {
		case <-doneCh:
			timer.Stop()
			return
		case <-timer.C:
			timer.Reset(failperiod)
			if id >= 0 {
				cfg.StartServer(id)
				cfg.Connect(id, cfg.All())
			}
			id = rand.Intn(5)
			cfg.ShutdownServer(id)
		}
	}
}

func stastic(opCh, doneCh chan bool) {
	timer := time.NewTimer(statperiod)
	cnt := 0
	for {
		select {
		case <-doneCh:
			timer.Stop()
			return
		case <-opCh:
			cnt++
		case <-timer.C:
			timer.Reset(statperiod)
			fmt.Printf("Timestamp: %v, ops since last stamp: %v\n", time.Now(), cnt)
			cnt = 0
		}
	}
}
