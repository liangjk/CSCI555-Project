package client

import (
	comm "CSCI555Project/common"
	"CSCI555Project/config"
	"fmt"
	"log"
	"net/rpc"
	"time"
)

type clientSession struct {
	clientID      string
	serverAddress string
	startTime     time.Time
	leaseLength   time.Duration
	rpcClient     *rpc.Client
}

func (c *clientSession) sendKeepAlive() {
	for {
		req := &comm.KeepAliveReq{
			ClientID: c.clientID,
		}
		res := &comm.KeepAliveRes{}
		err := c.rpcClient.Call("ChubbyServer.KeepAlive", req, res)
		if err != nil {
			log.Println("Error: failed to call KeepAlive rpc -> ", err)
			return
		} else {
			fmt.Println("Receive KeepAlive response from server")
			c.leaseLength = res.LeaseLength
			c.startTime = time.Now()
		}
	}
}

func (c *clientSession) keepAliveRoutine() {
	// client initiates a new KeepAlive immediately after receiving the previous reply
	go c.sendKeepAlive()
	for {
		leaseDeadline := c.startTime.Add(c.leaseLength)
		if time.Now().After(leaseDeadline) {
			fmt.Println("Session has expired")
			// jeopardy start: 45s
		}
		time.Sleep(time.Second) // check every second
	}
}

func (c *clientSession) dialServer(ipAddress string) bool {
	client, err := rpc.Dial("tcp", ipAddress)
	if err != nil {
		log.Printf("Error: Cannot connect to server at %s -> %v\n", ipAddress, err)
		return false
	}
	c.rpcClient = client
	return true
}

func (c *clientSession) findMaster() {
	serverList := config.ReadServerList("./config/server-list.json")
	for _, server := range serverList {
		if c.dialServer(server.IPAdress) {
			res := &comm.StartSessionRes{}
			req := &comm.StartSessionReq{
				ClientID: c.clientID,
			}
			if err := c.rpcClient.Call("ChubbyServer.StartSession", req, res); err != nil {
				log.Printf("Error: failed to call StartSession RPC at %s -> %v\n", server.IPAdress, err)
			} else if res.IsMaster == false {
				// call rpc again to the master node
				if c.dialServer(res.ServerAddress) {
					if err := c.rpcClient.Call("ChubbyServer.StartSession", req, res); err != nil {
						log.Printf("Error: failed to call StartSession RPC at %s -> %v\n", res.ServerAddress, err)
					} else {
						// successfully found the master and created a session on master
						c.serverAddress = res.ServerAddress
						c.leaseLength = res.LeaseLength
						go c.keepAliveRoutine()
						break
					}
				}
			} else {
				// current server is the master node
				c.serverAddress = res.ServerAddress
				c.leaseLength = res.LeaseLength
				go c.keepAliveRoutine()
				break
			}
		}
	}
}

func StartClient() {
	session := &clientSession{
		clientID:  "0",
		startTime: time.Now(),
	}
	session.findMaster()
}
