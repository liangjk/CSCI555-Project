package main

import (
	"CSCI555Project/client"
	"CSCI555Project/config"
	"CSCI555Project/server"
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	inputChan := make(chan string)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("> ")
			text, _ := reader.ReadString('\n')
			inputChan <- text
		}
	}()
	for {
		input := <-inputChan
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}
		if parts[0] == "startServer" {
			conf := config.ReadConfig(parts[1]) // ./config/server-1.json
			server.StartServer(conf)
		} else if parts[0] == "startClient" {
			client.StartClient()
		}
	}

}
