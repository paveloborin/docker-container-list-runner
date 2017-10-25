package main

import (
	"fmt"
	"github.com/paveloborin/docker-container-list-runner/dockerRunner"
	"time"
)

func main() {

	dockerRunner, error := dockerRunner.New()
	if nil != error {
		fmt.Errorf("error %s", error)
	}

	dockerRunner.Run()
	time.Sleep(30*time.Second)
	dockerRunner.Stop()
}
