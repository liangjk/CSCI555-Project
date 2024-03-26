package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	NodeType   string
	IPAdress   string
	ServerList []string
}

type ServerElement struct {
	IPAdress string
}

func ReadConfig(datapath string) *Config {
	jsonData, err := os.ReadFile(datapath)
	if err != nil {
		log.Println("Error: Unable to read the file -> ", err)
		return nil
	}
	var conf Config
	if err := json.Unmarshal(jsonData, &conf); err != nil {
		log.Println("Error: Unable to parse the JSON file -> ", err)
		return nil
	}
	return &conf
}

func ReadServerList(datapath string) []ServerElement {
	jsonData, err := os.ReadFile(datapath)
	if err != nil {
		log.Println("Error: Unable to read the file -> ", err)
		return nil
	}
	var serverList []ServerElement
	if err := json.Unmarshal(jsonData, &serverList); err != nil {
		log.Println("Error: Unable to parse the JSON file -> ", err)
		return nil
	}
	return serverList
}
