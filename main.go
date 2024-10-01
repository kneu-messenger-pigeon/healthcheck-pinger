package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const contentType = "text/plain"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	composeProjectName := os.Getenv("COMPOSE_PROJECT_NAME")
	_, _ = fmt.Fprintln(os.Stdout, "COMPOSE_PROJECT_NAME: "+composeProjectName)

	timeInterval, _ := strconv.Atoi(os.Getenv("INTERVAL"))
	if timeInterval <= 0 {
		timeInterval = 60
	}
	ticker := time.NewTicker(time.Duration(timeInterval) * time.Second)

	healthCheckSuccessPingUrl := os.Getenv("HEALTHCHECK_PING_URL")
	healthCheckErrorPingUrl := strings.TrimRight(healthCheckSuccessPingUrl, "/") + "/fail"

	_, _ = fmt.Fprintln(os.Stdout, "HEALTHCHECK_PING_URL: "+healthCheckSuccessPingUrl)

	ignoreServices := parseIgnoreServices(os.Getenv("IGNORE_SERVICES"))
	_, _ = fmt.Fprintln(os.Stdout, "IGNORE_SERVICES: "+strings.Join(ignoreServices, ", "))

	errorThreshold, _ := strconv.Atoi(os.Getenv("ERROR_THRESHOLD"))

	httpClient := &http.Client{
		Timeout: time.Second * 5,
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(ctx)

	var containers []types.Container
	var dockerContainer types.Container

	options := container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(),
	}

	options.Filters.Add("label", "com.docker.compose.project="+composeProjectName)

	runningServices := make([]string, 0)
	exitedServices := make([]string, 0)
	unhealthyServices := make([]string, 0)

	var errorString string
	errorCount := 0
	for {
		containers, err = cli.ContainerList(ctx, options)

		runningServices = make([]string, 0, len(containers))
		exitedServices = make([]string, 0, len(containers))
		unhealthyServices = make([]string, 0, len(containers))

		for _, dockerContainer = range containers {
			if ignoreServices.Contains(dockerContainer.Labels["com.docker.compose.service"]) {
				continue
			} else if strings.Contains(dockerContainer.Status, "unhealthy") {
				unhealthyServices = append(unhealthyServices, dockerContainer.Labels["com.docker.compose.service"])
			} else if dockerContainer.State == "running" {
				runningServices = append(runningServices, dockerContainer.Labels["com.docker.compose.service"])
			} else {
				exitedServices = append(exitedServices, dockerContainer.Labels["com.docker.compose.service"])
			}
		}

		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}

		if err == nil && len(runningServices) > 0 && len(unhealthyServices) == 0 && len(exitedServices) == 0 {
			errorCount = 0
			_, err = httpClient.Post(
				healthCheckSuccessPingUrl, contentType,
				strings.NewReader(strings.Join(runningServices, "\n")),
			)
		} else {
			errorCount++

			errorString = fmt.Sprintf("%v", err) + "\n" +
				strings.Join(exitedServices, "\n") + "\n" +
				strings.Join(unhealthyServices, "\n")

			errorString = strings.Trim(
				strings.Replace(errorString, "<nil>\n", "", 1),
				"\n",
			)

			if errorCount >= errorThreshold {
				_, err = httpClient.Post(
					healthCheckErrorPingUrl, contentType,
					strings.NewReader(errorString),
				)
			}
		}

		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}

		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
		}
	}
}

type IgnoreServices []string

func parseIgnoreServices(value string) (ignoreServices IgnoreServices) {
	for _, item := range strings.Split(
		strings.Replace(
			strings.Trim(value, " ,;"),
			";", ",", -1,
		),
		",",
	) {
		item = strings.Trim(item, " ")
		if item != "" {
			ignoreServices = append(ignoreServices, item)
		}
	}
	return
}

func (ignoreServices *IgnoreServices) Contains(value string) bool {
	for _, item := range *ignoreServices {
		if item == value {
			return true
		}
	}
	return false
}
