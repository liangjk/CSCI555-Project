package main

import (
	"fmt"
	"math/rand"
	"time"
)

func noFailure(period time.Duration, cfg *Config, doneCh chan bool) {
	timer := time.NewTimer(period)
	for {
		select {
		case <-doneCh:
			timer.Stop()
			return
		case <-timer.C:
			timer.Reset(period)
		}
	}
}

func periodicalLeaderFail(period time.Duration, cfg *Config, doneCh chan bool) {
	timer := time.NewTimer(period)
	id := -1
	for {
		select {
		case <-doneCh:
			timer.Stop()
			return
		case <-timer.C:
			timer.Reset(period)
			if id >= 0 {
				cfg.StartServer(id)
				cfg.Connect(id, cfg.All())
			}
			id = cfg.Leader()
			cfg.ShutdownServer(id)
		}
	}
}

func nonLeaderFail(period time.Duration, cfg *Config, doneCh chan bool) {
	ld := cfg.Leader()
	srvs := make([]int, 0)
	for i := 0; i < 5; i++ {
		if i != ld {
			srvs = append(srvs, i)
		}
	}
	timer := time.NewTimer(period / 2)
	updown := false
	id := 0
	for {
		select {
		case <-doneCh:
			timer.Stop()
			return
		case <-timer.C:
			timer.Reset(period / 2)
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

func randomFail(period time.Duration, cfg *Config, doneCh chan bool) {
	timer := time.NewTimer(period / 4)
	id := -1
	for {
		select {
		case <-doneCh:
			timer.Stop()
			return
		case <-timer.C:
			timer.Reset(period / 4)
			if id >= 0 {
				cfg.StartServer(id)
				cfg.Connect(id, cfg.All())
			}
			id = rand.Intn(5)
			cfg.ShutdownServer(id)
		}
	}
}

func stastic(period time.Duration, opCh, doneCh chan bool) {
	timer := time.NewTimer(period)
	cnt := 0
	for {
		select {
		case <-doneCh:
			timer.Stop()
			return
		case <-opCh:
			cnt++
		case <-timer.C:
			timer.Reset(period)
			fmt.Printf("Timestamp: %v, ops since last stamp: %v\n", time.Now(), cnt)
			cnt = 0
		}
	}
}
