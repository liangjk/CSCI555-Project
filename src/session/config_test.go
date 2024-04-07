package session_test

import (
	"CSCI555Project/labrpc"
	"CSCI555Project/paxosserver"
	"CSCI555Project/raftserver"
	"CSCI555Project/session"
	"encoding/base64"
	"fmt"
	"sync"
	"testing"
	"time"

	crand "crypto/rand"
)

func randstring(n int) string {
	b := make([]byte, 2*n)
	crand.Read(b)
	s := base64.URLEncoding.EncodeToString(b)
	return s[0:n]
}

type Servers interface {
	Kill()
	Shutdown(int, *sync.Mutex)
	Start([]*labrpc.ClientEnd, int)
	Service(int) (*labrpc.Service, *labrpc.Service)
	Leader() int
}

type Config struct {
	mu       sync.Mutex
	t        *testing.T
	n        int
	protocol session.Protocol

	net                 *labrpc.Network
	reliable, longdelay bool
	endnames            [][]string
	sessions            map[*session.Session][]string

	servers Servers

	start time.Time

	t0     time.Time
	rpcs0  int
	bytes0 int64
}

func (cfg *Config) makeServers() {
	if cfg.protocol == session.Raft {
		cfg.servers = raftserver.MakeRaftServers(cfg.n)
	} else {
		cfg.servers = paxosserver.MakePaxosServers(cfg.n)
	}
}

func (cfg *Config) SetNetwork(unreliable bool, longdelay bool) {
	cfg.net.Reliable(!unreliable)
	cfg.reliable = !unreliable
	cfg.net.LongDelays(longdelay)
	cfg.net.LongReordering(longdelay)
	cfg.longdelay = longdelay
}

func MakeConfig(t *testing.T, n int, unreliable bool, longdelay bool, protocol session.Protocol) *Config {
	cfg := &Config{t: t, n: n, protocol: protocol}

	cfg.net = labrpc.MakeNetwork()
	cfg.endnames = make([][]string, cfg.n)
	cfg.sessions = make(map[*session.Session][]string)

	cfg.makeServers()

	cfg.start = time.Now()

	for i := 0; i < cfg.n; i++ {
		cfg.StartServer(i)
	}

	cfg.ConnectAll()
	cfg.SetNetwork(unreliable, longdelay)

	return cfg
}

func (cfg *Config) Cleanup() {
	cfg.mu.Lock()
	cfg.servers.Kill()
	cfg.net.Cleanup()
	cfg.mu.Unlock()
}

// attach server i to servers listed in to
// caller must hold cfg.mu
func (cfg *Config) connectUnlocked(i int, to []int) {
	// log.Printf("connect peer %d to %v\n", i, to)

	// outgoing socket files
	for j := 0; j < len(to); j++ {
		endname := cfg.endnames[i][to[j]]
		cfg.net.Enable(endname, true)
	}

	// incoming socket files
	for j := 0; j < len(to); j++ {
		endname := cfg.endnames[to[j]][i]
		cfg.net.Enable(endname, true)
	}
}

func (cfg *Config) Connect(i int, to []int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.connectUnlocked(i, to)
}

// detach server i from the servers listed in from
// caller must hold cfg.mu
func (cfg *Config) disconnectUnlocked(i int, from []int) {
	// log.Printf("disconnect peer %d from %v\n", i, from)

	// outgoing socket files
	for j := 0; j < len(from); j++ {
		if cfg.endnames[i] != nil {
			endname := cfg.endnames[i][from[j]]
			cfg.net.Enable(endname, false)
		}
	}

	// incoming socket files
	for j := 0; j < len(from); j++ {
		if cfg.endnames[j] != nil {
			endname := cfg.endnames[from[j]][i]
			cfg.net.Enable(endname, false)
		}
	}
}

func (cfg *Config) Disconnect(i int, from []int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.disconnectUnlocked(i, from)
}

func (cfg *Config) All() []int {
	all := make([]int, cfg.n)
	for i := 0; i < cfg.n; i++ {
		all[i] = i
	}
	return all
}

func (cfg *Config) ConnectAll() {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	for i := 0; i < cfg.n; i++ {
		cfg.connectUnlocked(i, cfg.All())
	}
}

// Sets up 2 partitions with connectivity between servers in each  partition.
func (cfg *Config) Partition(p1 []int, p2 []int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	// log.Printf("partition servers into: %v %v\n", p1, p2)
	for i := 0; i < len(p1); i++ {
		cfg.disconnectUnlocked(p1[i], p2)
		cfg.connectUnlocked(p1[i], p1)
	}
	for i := 0; i < len(p2); i++ {
		cfg.disconnectUnlocked(p2[i], p1)
		cfg.connectUnlocked(p2[i], p2)
	}
}

func (cfg *Config) MakeSession(to []int) *session.Session {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	ends := make([]*labrpc.ClientEnd, cfg.n)
	endnames := make([]string, cfg.n)
	for j := 0; j < cfg.n; j++ {
		endnames[j] = randstring(20)
		ends[j] = cfg.net.MakeEnd(endnames[j])
		cfg.net.Connect(endnames[j], j)
	}

	var threshold int64 = 2
	if !cfg.reliable {
		threshold += 2
	}
	if cfg.longdelay {
		threshold += 4
	}
	sess := session.Init(ends, cfg.protocol, threshold)
	cfg.sessions[sess] = endnames
	cfg.connectSessionUnlocked(sess, to)
	return sess
}

func (cfg *Config) DeleteSession(sess *session.Session) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	for _, end := range cfg.sessions[sess] {
		cfg.net.DeleteEnd(end)
	}
	delete(cfg.sessions, sess)
}

// caller should hold cfg.mu
func (cfg *Config) connectSessionUnlocked(sess *session.Session, to []int) {
	endnames := cfg.sessions[sess]
	for j := 0; j < len(to); j++ {
		s := endnames[to[j]]
		cfg.net.Enable(s, true)
	}
}

func (cfg *Config) ConnectSession(sess *session.Session, to []int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.connectSessionUnlocked(sess, to)
}

// caller should hold cfg.mu
func (cfg *Config) disconnectSessionUnlocked(sess *session.Session, from []int) {
	// log.Printf("DisconnectClient %v from %v\n", ck, from)
	endnames := cfg.sessions[sess]
	for j := 0; j < len(from); j++ {
		s := endnames[from[j]]
		cfg.net.Enable(s, false)
	}
}

func (cfg *Config) DisconnectSession(sess *session.Session, from []int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.disconnectSessionUnlocked(sess, from)
}

// Shutdown a server by isolating it
func (cfg *Config) ShutdownServer(i int) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	cfg.disconnectUnlocked(i, cfg.All())

	cfg.net.DeleteServer(i)

	cfg.servers.Shutdown(i, &cfg.mu)
}

func (cfg *Config) StartServer(i int) {
	cfg.mu.Lock()

	// a fresh set of outgoing ClientEnd names.
	cfg.endnames[i] = make([]string, cfg.n)
	for j := 0; j < cfg.n; j++ {
		cfg.endnames[i][j] = randstring(20)
	}

	// a fresh set of ClientEnds.
	ends := make([]*labrpc.ClientEnd, cfg.n)
	for j := 0; j < cfg.n; j++ {
		ends[j] = cfg.net.MakeEnd(cfg.endnames[i][j])
		cfg.net.Connect(cfg.endnames[i][j], j)
	}

	cfg.servers.Start(ends, i)
	cfg.mu.Unlock()

	srv := labrpc.MakeServer()
	csvc, psvc := cfg.servers.Service(i)
	srv.AddService(csvc)
	srv.AddService(psvc)
	cfg.net.AddServer(i, srv)
}

func (cfg *Config) Leader() int {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	ld := cfg.servers.Leader()
	for ld < 0 {
		cfg.mu.Unlock()
		time.Sleep(time.Microsecond * 1000)
		cfg.mu.Lock()
		ld = cfg.servers.Leader()
	}
	return ld
}

// start a Test.
// print the Test message.
func (cfg *Config) begin(description string) {
	fmt.Printf("%s ...\n", description)
	cfg.t0 = time.Now()
	cfg.rpcs0 = cfg.net.GetTotalCount()
	cfg.bytes0 = cfg.net.GetTotalBytes()
}

// end a Test -- the fact that we got here means there
// was no failure.
// print the Passed message,
// and some performance numbers.
func (cfg *Config) end() {
	if cfg.t.Failed() == false {
		t := time.Since(cfg.t0).Seconds() // real time
		npeers := cfg.n
		nrpc := cfg.net.GetTotalCount() - cfg.rpcs0
		nbytes := cfg.net.GetTotalBytes() - cfg.bytes0

		fmt.Printf("  ... Passed --")
		fmt.Printf("  %4.1f  %d %5d %6.3fKB\n", t, npeers, nrpc, float32(nbytes)/1024)
	}
}
