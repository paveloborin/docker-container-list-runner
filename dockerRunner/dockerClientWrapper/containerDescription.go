package dockerClientWrapper

type ContainerDescription struct {
	Name       string
	Env        map[string]string
	DockerPort int
	HostPort   int
	ID         string
}
