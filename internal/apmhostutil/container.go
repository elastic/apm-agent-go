package apmhostutil

// ContainerID returns the full container ID in which the process is executing,
// or an error if the container ID could not be determined.
func ContainerID() (string, error) {
	return DockerContainerID()
}

// DockerContainerID returns the full Docker container ID in which the process
// is executing, or an error if the container ID could not be determined.
func DockerContainerID() (string, error) {
	return dockerContainerID()
}
