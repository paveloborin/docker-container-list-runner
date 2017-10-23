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
)

/*type dockerClientInterface interface {
	ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error)
	ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error
	ContainerWait(ctx context.Context, containerID string) (int64, error)
	ContainerExport(ctx context.Context, containerID string) (io.ReadCloser, error)
	ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
}*/

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

func (c *DockerClientWrapper) StartContainer(container ContainerDescription, wg *sync.WaitGroup, done <-chan bool, runningContainersChan chan ContainerDescription) {
	defer wg.Done()
	select {
	case <-done:
		return
	default:

		containerCreatedResponse := createContainer(&c.dockerClient, container)
		container.ID = containerCreatedResponse.ID
		fmt.Printf("Start container %s %s \n", container.ID, container.Name)

		// start container
		if err := c.dockerClient.ContainerStart(
			context.TODO(),
			container.ID,
			types.ContainerStartOptions{},
		); nil != err {
			panic(err)
		}
		runningContainersChan <- container

	}
}

func (c *DockerClientWrapper) StopContainer(container ContainerDescription, wg *sync.WaitGroup, done <-chan bool, stoppedContainersChan chan ContainerDescription) {
	defer wg.Done()

	select {
	case <-done:
		return
	default:
		fmt.Printf("Stop container %s %s\n", container.ID, container.Name)
		duration := time.Duration(5 * time.Second)
		err := c.dockerClient.ContainerStop(context.TODO(), container.ID, &duration)
		if nil != err {
			fmt.Errorf("stop container error %s \n", err)
		}
		stoppedContainersChan <- container
	}

}

func createContainer(dockerClient *client.Client, description ContainerDescription) container.ContainerCreateCreatedBody {
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

		err = nil
		fmt.Printf("unable to find image '%s' locally\n", imageName)

		_, err = dockerClient.ImagePull(context.TODO(), imageName, types.ImagePullOptions{})
		if err != nil {
			panic(err)
		}

		// retry create
		containerCreatedResponse, err := dockerClient.ContainerCreate(
			context.TODO(),
			containerConfig,
			hostConfig,
			nil,
			"",
		)
		if nil != err {
			panic(err)
		}
		fmt.Println(containerCreatedResponse.ID)

	}

	return containerCreatedResponse
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
