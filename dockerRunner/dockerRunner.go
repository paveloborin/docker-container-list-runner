package dockerRunner

import (
	"log"
	"sync"
)

type DockerRunner struct {
	dockerClient          DockerClientWrapper
	initContainersChan    <-chan ContainerDescription
	runningContainersChan chan ContainerDescription
	stoppedContainersChan chan ContainerDescription
	done                  chan bool
}

func New() (*DockerRunner, error) {
	dockerClient, err := NewDockerClientWrapper()
	if err != nil {
		return nil, err
	}

	initContainersChan, runningContainersChan, stoppedContainersChan := InitChannels()

	return &DockerRunner{
		dockerClient:          *dockerClient,
		initContainersChan:    initContainersChan,
		runningContainersChan: runningContainersChan,
		stoppedContainersChan: stoppedContainersChan,
		done:                  make(chan bool),
	}, nil
}

func InitChannels() (<-chan ContainerDescription, chan ContainerDescription, chan ContainerDescription) {

	inArray := LoadConfiguration()
	outChan := make(chan ContainerDescription, len(inArray))
	go func() {
		defer close(outChan)
		for _, value := range inArray {
			outChan <- value
		}
	}()

	return outChan, make(chan ContainerDescription, len(inArray)), make(chan ContainerDescription, len(inArray))
}

func (dockerRunner *DockerRunner) Run() {
	var wg sync.WaitGroup

	defer close(dockerRunner.runningContainersChan)
	log.Println("Start containers running")

	for containerConfiguration := range dockerRunner.initContainersChan {
		wg.Add(1)
		go func(containerConfiguration ContainerDescription) {
			dockerRunner.dockerClient.StartContainer(containerConfiguration, &wg, dockerRunner.done, dockerRunner.runningContainersChan)
		}(containerConfiguration)
	}

	wg.Wait()
	log.Println("All containers has runned")
}

func (dockerRunner *DockerRunner) Stop() {
	var wg sync.WaitGroup

	defer close(dockerRunner.stoppedContainersChan)
	log.Println("Stop all running containers")

	for containerConfiguration := range dockerRunner.runningContainersChan {
		wg.Add(1)
		go func(containerConfiguration ContainerDescription) {
			dockerRunner.dockerClient.StopContainer(containerConfiguration, &wg, dockerRunner.stoppedContainersChan)
		}(containerConfiguration)
	}

	wg.Wait()
	log.Println("All containers has stopped")
}
