package dockerRunner

import (
	"os"
	"fmt"
	"encoding/json"
)

type ContainerDescription struct {
	Name       string `json:"Name"`
	Env        map[string]string `json:"Env"`
	DockerPort int `json:"DockerPort"`
	HostPort   int `json:"HostPort"`
	ID         string
}


var config []ContainerDescription

func LoadConfiguration() []ContainerDescription {

	file := "./containerConfig.json"
	configFile, err := os.Open(file)

	if err != nil {
		fmt.Println(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config
}
