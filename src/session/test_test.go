package session_test

import (
	"CSCI555Project/session"
	"sync"
	"testing"
	"time"
)

func (cfg *Config) check(res session.Err, exp []session.Err) bool {
	for _, c := range exp {
		if res == c {
			return true
		}
	}
	cfg.t.Fatalf("Expected: %v, got :%v\n", exp, res)
	return false
}

func basic(cfg *Config) {
	sess := cfg.MakeSession(cfg.All())
	path := "basic/mu"
	cfg.check(sess.Acquire(path), []session.Err{session.LockNotExist})
	cfg.check(sess.Create(path), []session.Err{session.OK})
	cfg.check(sess.Acquire(path), []session.Err{session.OK})
	cfg.check(sess.Acquire(path), []session.Err{session.LockReacquire})
	if !sess.IsHolding(path) {
		cfg.t.Fatalf("Not holding lock!\n")
	}
	cfg.check(sess.Remove(path), []session.Err{session.LockBusy})
	cfg.check(sess.Release(path), []session.Err{session.OK})
	cfg.check(sess.Remove(path), []session.Err{session.OK})
}

func TestBasic(t *testing.T) {
	rfcfg := MakeConfig(t, 5, false, false, session.Raft)
	defer rfcfg.Cleanup()

	rfcfg.begin("Test: Basic Raft 5 Servers")
	basic(rfcfg)
	rfcfg.end()

	pxcfg := MakeConfig(t, 5, false, false, session.Paxos)
	defer pxcfg.Cleanup()

	pxcfg.begin("Test: Basic Paxos 5 Servers")
	basic(pxcfg)
	pxcfg.end()
}

func contend(cfg *Config) {
	sess1 := cfg.MakeSession(cfg.All())
	sess2 := cfg.MakeSession(cfg.All())
	path := "contend/mu"
	cfg.check(sess1.Acquire(path), []session.Err{session.LockNotExist})
	cfg.check(sess2.Acquire(path), []session.Err{session.LockNotExist})
	cfg.check(sess1.Create(path), []session.Err{session.OK})
	for {
		code := sess2.Acquire(path)
		cfg.check(code, []session.Err{session.OK, session.LockNotExist})
		if code == session.OK {
			break
		}
	}
	cfg.check(sess1.Acquire(path), []session.Err{session.LockBusy})
	cfg.check(sess1.Release(path), []session.Err{session.LockNotAcquired})
	cfg.check(sess2.Release(path), []session.Err{session.OK})
	for {
		code := sess1.Acquire(path)
		cfg.check(code, []session.Err{session.OK, session.LockBusy})
		if code == session.OK {
			break
		}
	}
	cfg.check(sess2.Acquire(path), []session.Err{session.LockBusy})
	cfg.check(sess2.Release(path), []session.Err{session.LockNotAcquired})
	cfg.check(sess1.Release(path), []session.Err{session.OK})
	for {
		code := sess2.Remove(path)
		cfg.check(code, []session.Err{session.OK, session.LockBusy})
		if code == session.OK {
			break
		}
	}
}

func TestContend(t *testing.T) {
	rfcfg := MakeConfig(t, 5, false, false, session.Raft)
	defer rfcfg.Cleanup()

	rfcfg.begin("Test: Contend Raft 5 Servers")
	contend(rfcfg)
	rfcfg.end()

	pxcfg := MakeConfig(t, 5, false, false, session.Paxos)
	defer pxcfg.Cleanup()

	pxcfg.begin("Test: Contend Paxos 5 Servers")
	contend(pxcfg)
	pxcfg.end()
}

func concurrent(cfg *Config) {
	clientNum := 20
	sess := make([]*session.Session, clientNum)
	res := make([]bool, clientNum)
	fail := make([]bool, clientNum)
	path := "concurrent/mu"
	wg := sync.WaitGroup{}
	wg.Add(clientNum)
	for i := 0; i < clientNum; i++ {
		num := i
		go func() {
			defer wg.Done()
			sess[num] = cfg.MakeSession(cfg.All())
			if !cfg.check(sess[num].Acquire(path), []session.Err{session.LockNotExist}) {
				fail[num] = true
				return
			}
		}()
	}
	wg.Wait()
	wg.Add(clientNum)
	for i := 0; i < clientNum; i++ {
		num := i
		go func() {
			defer wg.Done()
			if !cfg.check(sess[num].Create(path), []session.Err{session.OK}) {
				fail[num] = true
				return
			}
			if !cfg.check(sess[num].Acquire(path), []session.Err{session.OK, session.LockBusy}) {
				fail[num] = true
				return
			}
			res[num] = sess[num].IsHolding(path)
		}()
	}
	wg.Wait()
	wg.Add(clientNum)
	for i := 0; i < clientNum; i++ {
		num := i
		go func() {
			defer wg.Done()
			if !cfg.check(sess[num].Release(path), []session.Err{session.OK, session.LockNotAcquired}) {
				fail[num] = true
				return
			}
			if !cfg.check(sess[num].Remove(path), []session.Err{session.OK, session.LockNotAcquired, session.LockBusy}) {
				fail[num] = true
				return
			}
		}()
	}
	wg.Wait()
	cnt := 0
	for i := 0; i < clientNum; i++ {
		if fail[i] {
			cfg.t.Fatalf("Session %v failed\n", i)
		}
		if res[i] {
			cnt++
		}
	}
	if cnt != 1 {
		cfg.t.Fatalf("Expected 1 lock, got %v\n", cnt)
	}
	cfg.check(sess[0].Acquire(path), []session.Err{session.LockNotExist})
}

func TestConcurrent(t *testing.T) {
	rfcfg := MakeConfig(t, 5, false, false, session.Raft)
	defer rfcfg.Cleanup()

	rfcfg.begin("Test: Concurrent Raft 5 Servers")
	concurrent(rfcfg)
	rfcfg.end()

	pxcfg := MakeConfig(t, 5, false, false, session.Paxos)
	defer pxcfg.Cleanup()

	pxcfg.begin("Test: Concurrent Paxos 5 Servers")
	concurrent(pxcfg)
	pxcfg.end()
}

func leaseexpire(cfg *Config) {
	sess := cfg.MakeSession(cfg.All())
	path := "lease/mu"
	cfg.check(sess.Create(path), []session.Err{session.OK})
	cfg.check(sess.Acquire(path), []session.Err{session.OK})
	time.Sleep(time.Second * 20)
	if !sess.IsHolding(path) {
		cfg.t.Fatalf("Lock lost")
	}
	cfg.DeleteSession(sess)
	sess = cfg.MakeSession(cfg.All())
	cfg.check(sess.Acquire(path), []session.Err{session.LockBusy})
	time.Sleep(time.Second * 20)
	cfg.check(sess.Acquire(path), []session.Err{session.OK})
	cfg.check(sess.Release(path), []session.Err{session.OK})
}

func TestLease(t *testing.T) {
	rfcfg := MakeConfig(t, 5, false, false, session.Raft)
	defer rfcfg.Cleanup()

	rfcfg.begin("Test: Lease Expire Raft 5 Servers")
	leaseexpire(rfcfg)
	rfcfg.end()

	pxcfg := MakeConfig(t, 5, false, false, session.Paxos)
	defer pxcfg.Cleanup()

	pxcfg.begin("Test: Lease Expire Paxos 5 Servers")
	leaseexpire(pxcfg)
	pxcfg.end()
}

func TestUnreliable(t *testing.T) {
	rfcfg := MakeConfig(t, 5, true, true, session.Raft)
	defer rfcfg.Cleanup()

	rfcfg.begin("Test: Unreliable and LongDelay Raft 5 Servers")
	basic(rfcfg)
	contend(rfcfg)
	concurrent(rfcfg)
	leaseexpire(rfcfg)
	rfcfg.end()

	pxcfg := MakeConfig(t, 5, false, false, session.Paxos)
	defer pxcfg.Cleanup()

	pxcfg.begin("Test: Unreliable and LongDelay Paxos 5 Servers")
	basic(pxcfg)
	contend(pxcfg)
	concurrent(pxcfg)
	leaseexpire(pxcfg)
	pxcfg.end()
}
