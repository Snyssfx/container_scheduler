package containers

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// docker is a controller for starting and stopping docker containers using
// command line.
type docker struct {
	l                   *zap.SugaredLogger
	imageName, imageTag string
	port                int
	name                string
	envs                [][]string
}

func newDocker(
	logger *zap.SugaredLogger,
	imageName, imageTag string,
	port int, name string,
	envs [][]string,
) *docker {
	return &docker{
		l:         logger,
		imageName: imageName,
		imageTag:  imageTag,
		port:      port,
		name:      name,
		envs:      envs,
	}
}

// Run creates and starts the docker container.
func (d *docker) Run() error {
	err := runCmd(d.getRunString())
	if err != nil {
		return fmt.Errorf("cannot run docker container: %w", err)
	}

	d.l.Infof("ran docker container.")
	return nil
}

func (d *docker) getRunString() string {
	var pairs []string
	for _, kv := range d.envs {
		pairs = append(pairs, strings.Join(kv, "="))
	}
	environs := strings.Join(pairs, " ")

	return fmt.Sprintf(
		"run --detach --publish %d:8080 --env %s --name %s %s:%s",
		d.port, environs, d.name, d.imageName, d.imageTag)
}

// Stop stops the container and remove it.
func (d *docker) Stop() error {
	err := runCmd(fmt.Sprintf("stop %s", d.name))
	if err != nil {
		return fmt.Errorf("cannot stop docker container %q: %w", d.name, err)
	}

	err = runCmd(fmt.Sprintf("rm %s", d.name))
	if err != nil {
		return fmt.Errorf("cannot rm docker container %q: %w", d.name, err)
	}

	return nil
}

func runCmd(cmdStr string) error {
	cmd := exec.Command("docker", strings.Split(cmdStr, " ")...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("cannot start cmd exec: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("cannot end cmd exec: %w", err)
	}

	return nil
}
