# docker-container-list-runner
Docker client wrapper providing simple way for running containers from list.

**Run:**
dep ensure

**Config**
Docker containers configuration file: _containerConfig.json_

**Using**

Create docker client: `dockerRunner, error := dockerRunner.New()`

Run all docker containers from the config file: `dockerRunner.Run()`

Stop containers that we ran: `dockerRunner.Stop()`