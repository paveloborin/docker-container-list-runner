package dockerRunner

import (
	"github.com/paveloborin/docker-container-list-runner/dockerRunner/dockerClientWrapper"
	"sync"
	"fmt"
)

//три канала:
//1) контайнеры ожидающие запуска
//2) контанеры запущенные
//3) контанеры остановленные

//программа начинается с запуска всех контайнеров
//преобразование массива с данными о контенерах в канал
//процесс принимающий элемент из канала и запускающий его в горутине, которая в случае успеха добавляет элемент в канал запушенных контейнеров
//процесс остановки считывает элемент из канала стартовавших и запускат его остановку в горутине
type DockerRunner struct {
	dockerClient          dockerClientWrapper.DockerClientWrapper
	initContainersChan    <-chan dockerClientWrapper.ContainerDescription
	runningContainersChan chan dockerClientWrapper.ContainerDescription
	stoppedContainersChan chan dockerClientWrapper.ContainerDescription
	done                  chan bool
}

func New() (*DockerRunner, error) {
	cli, err := dockerClientWrapper.New()
	if err != nil {
		return nil, err
	}

	initChan, runningChan, stoppedChan := InitChans()

	return &DockerRunner{
		dockerClient:          *cli,
		initContainersChan:    initChan,
		done:                  make(chan bool),
		runningContainersChan: runningChan,
		stoppedContainersChan: stoppedChan,
	}, nil
}

/**
преобразование массива с параметрами контейнеров в канал initContainersChan, закрытие канала
 */
func InitChans() (<-chan dockerClientWrapper.ContainerDescription, chan dockerClientWrapper.ContainerDescription, chan dockerClientWrapper.ContainerDescription) {

	inArray := readContainersConfiguration()
	outChan := make(chan dockerClientWrapper.ContainerDescription, len(inArray))
	go func() {
		defer close(outChan)
		for _, value := range inArray {
			outChan <- value
		}
	}()
	return outChan, make(chan dockerClientWrapper.ContainerDescription, len(inArray)), make(chan dockerClientWrapper.ContainerDescription, len(inArray))
}

//TODO реализовать счиывание конфигурации из файла
func readContainersConfiguration() []dockerClientWrapper.ContainerDescription {
	containerDescription1 := dockerClientWrapper.ContainerDescription{Name: "percona:latest", Env: map[string]string{"MYSQL_ROOT_PASSWORD": "1"}, DockerPort: 3306, HostPort: 3306}
	containerDescription2 := dockerClientWrapper.ContainerDescription{Name: "percona:5.5.41", Env: map[string]string{"MYSQL_ROOT_PASSWORD": "1"}, DockerPort: 3307, HostPort: 3307}
	containerDescription3 := dockerClientWrapper.ContainerDescription{Name: "percona:5.6.26", Env: map[string]string{"MYSQL_ROOT_PASSWORD": "1"}, DockerPort: 3308, HostPort: 3308}

	return []dockerClientWrapper.ContainerDescription{containerDescription1, containerDescription2, containerDescription3}

}

/**
считывание из канала initContainersChan, запуск, запись в канал runningContainersChan, закрытие канала
 */
func (dockerRunner *DockerRunner) Run() {
	var wg sync.WaitGroup
	fmt.Println("start containers running")

	defer close(dockerRunner.runningContainersChan)

	for containerConfiguration := range dockerRunner.initContainersChan {
		wg.Add(1)
		go func(containerConfiguration dockerClientWrapper.ContainerDescription) {
			dockerRunner.dockerClient.StartContainer(containerConfiguration, &wg, dockerRunner.done, dockerRunner.runningContainersChan)
		}(containerConfiguration)
	}

	wg.Wait()
	fmt.Print("ended start\n\n")
}

/**
считывание из канала runningContainersChan, остановка контайнеров запись в канал stoppedContainersChan
 */
func (dockerRunner *DockerRunner) Stop() {
	var wg sync.WaitGroup
	fmt.Println("stop containers")

	defer close(dockerRunner.stoppedContainersChan)

	for containerConfiguration := range dockerRunner.runningContainersChan {
		wg.Add(1)
		go func(containerConfiguration dockerClientWrapper.ContainerDescription) {
			dockerRunner.dockerClient.StopContainer(containerConfiguration, &wg, dockerRunner.stoppedContainersChan)
		}(containerConfiguration)
	}

	wg.Wait()
	fmt.Print("ended stop\n\n")
}
