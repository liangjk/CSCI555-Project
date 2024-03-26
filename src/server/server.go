package server

import (
	comm "CSCI555Project/common"
	config "CSCI555Project/config"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"time"
)

type ChubbyServer struct {
	sessions   map[string]*serverSession // key: clientId, value: serverSession of that client
	address    string                    // IP address for current node
	serverlist []string                  // server address of other servers
}

// session for server only
type serverSession struct {
	lease     time.Duration
	startTime time.Time
	clientID  string
}

func (c *ChubbyServer) StartSession(req *comm.StartSessionReq, res *comm.StartSessionRes) error {
	session := &serverSession{
		lease:     12 * time.Second,
		startTime: time.Now(),
		clientID:  req.ClientID,
	}
	c.sessions[session.clientID] = session
	fmt.Println("successfully create a session")
	res.ServerAddress = "127.0.0.1:8000"
	res.LeaseLength = time.Second * 12
	return nil
}

func (c *ChubbyServer) KeepAlive(req *comm.KeepAliveReq, res *comm.KeepAliveRes) error {
	// after receiving KeepAlive, master should block RPC until the client's previous
	// lease interval is close to expiring
	session, exists := c.sessions[req.ClientID]
	if !exists {
		log.Println("Session does not exist!!!")
	}
	timeRemaining := session.startTime.Add(session.lease).Add(-1 * time.Second).Sub(time.Now())
	if timeRemaining > 0 {
		time.Sleep(timeRemaining)
	}
	// Update session with new lease
	session.startTime = time.Now()
	session.lease = 12 * time.Second
	res.LeaseLength = 12 * time.Second
	return nil
}

func StartServer(config *config.Config) {
	chubbyServer := &ChubbyServer{
		sessions:   make(map[string]*serverSession),
		address:    config.IPAdress,
		serverlist: config.ServerList,
	}
	ln, err := net.Listen("tcp", chubbyServer.address)
	if err != nil {
		log.Println("Error: cannot srart a TCP server -> ", err)
	}
	// serverHandler := new(ServerHandler)
	rpc.Register(chubbyServer)
	go rpc.Accept(ln)
	fmt.Println("accept listener")
}
