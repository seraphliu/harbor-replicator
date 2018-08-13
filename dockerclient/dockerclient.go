package dockerclient

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

const CmdTimeout = 300 * time.Second

type DockerCLI struct {
	registry string
	user     string
	pass     string
}

func NewDockerClient(registry string, user, pass string) *DockerCLI {
	return &DockerCLI{
		registry: registry,
		user:     user,
		pass:     pass,
	}
}

func execCommand(cmd *exec.Cmd, timeout time.Duration) error {
	if os.Getenv("verbose") == "1" {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return errors.Wrap(err, "get command stdout error")
		}
		go func() {
			io.Copy(os.Stdout, stdout)
		}()

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return errors.Wrap(err, "get command stderr error")
		}
		go func() {
			io.Copy(os.Stderr, stderr)
		}()
	}
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "execute command error")
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(timeout):
		if err := cmd.Process.Kill(); err != nil {
			return errors.Wrap(err, "failed to kill timeout process")
		}
		log.Println("process killed as timeout")
	case err := <-done:
		if err != nil {
			return fmt.Errorf("command finished with error = %v", err)
		}
	}
	return nil
}

func (d *DockerCLI) Login() error {
	cmd := exec.Command("docker", "login", "-u", d.user, "-p", d.pass, d.registry)
	return execCommand(cmd, 2 * time.Second)
}

func (d *DockerCLI) action(repo string, tag string, action string) error {
	name := fmt.Sprintf("%s/%s:%s", d.registry, repo, tag)
	log.Printf("%s image: %s\n", action, name)
	cmd := exec.Command("docker", action, name)

	timeout := CmdTimeout
	if v, ok := os.LookupEnv("timeout"); ok {
		if t, err := time.ParseDuration(v); err != nil {
			timeout = t
		}
	}
	return execCommand(cmd, timeout)
}

func (d *DockerCLI) Pull(repo string, tag string) error {
	return d.action(repo, tag, "pull")
}

func (d *DockerCLI) Push(repo string, tag string) error {
	return d.action(repo, tag, "push")
}

func (d *DockerCLI) Tag(oldRegistry string, repo string, tag string) error {
	oldName := fmt.Sprintf("%s/%s:%s", oldRegistry, repo, tag)
	newName := fmt.Sprintf("%s/%s:%s", d.registry, repo, tag)
	log.Printf("tag image: %s -> %s", oldName, newName)

	cmd := exec.Command("docker", "tag", oldName, newName)

	timeout := CmdTimeout
	if v, ok := os.LookupEnv("timeout"); ok {
		if t, err := time.ParseDuration(v); err != nil {
			timeout = t
		}
	}
	return execCommand(cmd, timeout)
}
