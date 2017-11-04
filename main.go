package main

import (
	"fmt"
	"github.com/paveloborin/docker-container-list-runner/dockerRunner"
	"time"
)

func main() {
	//Sample of using
	dockerRunner, error := dockerRunner.New()
	if nil != error {
		fmt.Errorf("error %s", error)
	}

	//Run all docker containers from the config file
	dockerRunner.Run()
	time.Sleep(30*time.Second)

	//Stop containers that we ran
	dockerRunner.Stop()
}
