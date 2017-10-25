package dockerClientWrapper

import (
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"sync"
	"fmt"
	"time"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"sort"
	"io"
	"io/ioutil"
)

type DockerClientWrapper struct {
	dockerClient client.Client
}

func New() (*DockerClientWrapper, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	if _, err := cli.Ping(context.Background()); err != nil {
		return nil, err
	}
	return &DockerClientWrapper{dockerClient: *cli}, nil
}

func (c *DockerClientWrapper) StartContainer(container ContainerDescription, wg *sync.WaitGroup, done chan bool, runningContainersChan chan ContainerDescription) {
	defer wg.Done()
	select {
	case <-done:
		return
	default:
		//create container
		containerCreatedResponse, err := createContainer(&c.dockerClient, container)
		if nil != err {
			fmt.Println(err)
			close(done)
			return
		}
		container.ID = containerCreatedResponse.ID
		fmt.Printf("Start container %s %s \n", container.ID, container.Name)

		// start container
		if err := c.dockerClient.ContainerStart(
			context.TODO(),
			container.ID,
			types.ContainerStartOptions{},
		); nil != err {
			close(done)
			fmt.Errorf("Error %s \n", err)
			return
		}
		runningContainersChan <- container

	}
}

func (c *DockerClientWrapper) StopContainer(container ContainerDescription, wg *sync.WaitGroup, stoppedContainersChan chan ContainerDescription) {
	defer wg.Done()

	fmt.Printf("Stop container %s %s\n", container.ID, container.Name)
	duration := time.Duration(5 * time.Second)
	err := c.dockerClient.ContainerStop(context.TODO(), container.ID, &duration)
	if nil != err {
		fmt.Errorf("stop container error %s \n", err)
	}
	stoppedContainersChan <- container

}

func createContainer(dockerClient *client.Client, description ContainerDescription) (container.ContainerCreateCreatedBody, error) {
	imageName := description.Name
	containerConfig, hostConfig := getContainerConfigs(imageName, description.HostPort, description.DockerPort, description.Env)
	containerCreatedResponse, err := dockerClient.ContainerCreate(
		context.TODO(),
		containerConfig,
		hostConfig,
		nil,
		"",
	)

	if nil != err {
		if !client.IsErrNotFound(err) {
			panic(err)
		}

		fmt.Printf("Unable to find image '%s' locally\n", imageName)
		fmt.Printf("Image '%s' pull, wait\n", imageName)

		resp, err := dockerClient.ImagePull(context.TODO(), imageName, types.ImagePullOptions{})
		if err != nil {
			fmt.Println(err)
			return container.ContainerCreateCreatedBody{}, err
		}
		io.Copy(ioutil.Discard, resp)

		// retry create
		containerCreatedResponse, err := dockerClient.ContainerCreate(
			context.TODO(),
			containerConfig,
			hostConfig,
			nil,
			"",
		)
		fmt.Println(err)

		return containerCreatedResponse, err

	}

	return containerCreatedResponse, nil
}

func getContainerConfigs(imageName string, hostPort, containerPort int, envVars map[string]string) (*container.Config, *container.HostConfig) {
	portBindings := nat.PortMap{}

	portMappings, err := nat.ParsePortSpec(fmt.Sprintf("%v:%v", hostPort, containerPort))
	if nil != err {
		panic(err)
	}

	for _, portMapping := range portMappings {
		if _, ok := portBindings[portMapping.Port]; ok {
			portBindings[portMapping.Port] = append(portBindings[portMapping.Port], portMapping.Binding)
		} else {
			portBindings[portMapping.Port] = []nat.PortBinding{portMapping.Binding}
		}
	}

	// construct container config
	containerConfig := &container.Config{
		Image: imageName,
		ExposedPorts: nat.PortSet{
			nat.Port("3306/tcp"): {},
		},
	}

	for envVarName, envVarValue := range envVars {
		containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%v=%v", envVarName, envVarValue))
	}
	// sort binds to make order deterministic; useful for testing
	sort.Strings(containerConfig.Env)
	//TODO
	for port := range portBindings {
		containerConfig.ExposedPorts[port] = struct{}{}
	}

	// construct host config
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Privileged:   true,
	}

	return containerConfig, hostConfig

}
