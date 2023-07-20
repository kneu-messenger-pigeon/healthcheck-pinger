package main

import (
	"errors"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestMainFunc(t *testing.T) {
	runWithDockerCompose := func(testCase string) {
		projectName := "test-" + testCase
		composeFile := "tests/docker-compose-" + testCase + ".yml"
		os.Setenv("COMPOSE_PROJECT_NAME", projectName)

		cmd := exec.Command(
			"docker", "compose",
			"-f", composeFile,
			"--project-name", projectName,
			"up", "-d",
		)

		err := cmd.Run()
		if err != nil {
			log.Panic(err)
		}

		time.Sleep(time.Second * 2)
		go func() {
			time.Sleep(time.Millisecond * 200)
			_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		}()

		main()

		cmd = exec.Command(
			"docker", "compose",
			"-f", composeFile,
			"--project-name", projectName,
			"down",
			"--remove-orphans", "--volumes",
			"--timeout", "1",
		)

		err = cmd.Run()
		if err != nil {
			log.Panic(err)
		}
	}

	t.Run("Wrong docker host", func(t *testing.T) {
		os.Setenv("DOCKER_HOST", "wronghost")
		defer os.Unsetenv("DOCKER_HOST")
		defer func() {
			assert.NotNil(t, recover())
		}()
		main()
	})

	t.Run("Unhealthy", func(t *testing.T) {
		mockHealthCheckIo := "https://mock-healthcheck.io/unhelathy-test/"
		os.Setenv("HEALTHCHECK_PING_URL", mockHealthCheckIo)
		gock.New(mockHealthCheckIo).
			Post("/fail").
			Times(1).
			BodyString("alpine-unhealthy").
			Reply(200)
		defer gock.Off()

		runWithDockerCompose("unhealthy")

		assert.True(t, gock.IsDone())
	})

	t.Run("Healthy", func(t *testing.T) {
		mockHealthCheckIo := "https://mock-healthcheck.io/healthy-test/"
		gock.New(mockHealthCheckIo).
			Post("/").
			Times(1).
			BodyString("alpine-healthy").
			Reply(200)
		defer gock.Off()

		os.Setenv("HEALTHCHECK_PING_URL", mockHealthCheckIo)

		runWithDockerCompose("healthy")

		assert.True(t, gock.IsDone())
	})

	t.Run("Exited", func(t *testing.T) {
		mockHealthCheckIo := "https://mock-healthcheck.io/exited-test/"
		gock.New(mockHealthCheckIo).
			Post("/fail").
			Times(1).
			BodyString("alpine-exited").
			Reply(200)
		defer gock.Off()

		os.Setenv("HEALTHCHECK_PING_URL", mockHealthCheckIo)
		os.Setenv("IGNORE_SERVICES", "alpine-unhealthy")

		runWithDockerCompose("exited")

		assert.True(t, gock.IsDone())
	})

	t.Run("Loop-and-empty-services", func(t *testing.T) {
		mockHealthCheckIo := "https://mock-healthcheck.io/loop/"
		gock.New(mockHealthCheckIo).
			Post("/fail").
			Times(2).
			BodyString("").
			Reply(0).SetError(errors.New("error"))

		defer gock.Off()

		os.Setenv("COMPOSE_PROJECT_NAME", "test-loop")
		os.Setenv("HEALTHCHECK_PING_URL", mockHealthCheckIo)
		os.Setenv("INTERVAL", "2")

		go func() {
			time.Sleep(time.Second * 3)
			_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		}()

		main()
		assert.True(t, gock.IsDone())
	})

	t.Run("Docker error", func(t *testing.T) {
		os.Setenv("DOCKER_HOST", "tcp://docker-host:2376")
		defer os.Unsetenv("DOCKER_HOST")

		mockHealthCheckIo := "https://mock-healthcheck.io/docker-error"
		gock.New(mockHealthCheckIo).
			Post("/fail").
			Times(1).
			BodyString("^error during connect.*").
			Reply(200)

		os.Setenv("COMPOSE_PROJECT_NAME", "test-docker-error")
		os.Setenv("HEALTHCHECK_PING_URL", mockHealthCheckIo)
		os.Setenv("INTERVAL", "10")

		go func() {
			time.Sleep(time.Millisecond * 200)
			_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		}()

		main()
		assert.True(t, gock.IsDone())
	})
}
