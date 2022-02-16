package docker

import (
	"context"
	"testing"

	"github.com/LambdaTest/synapse/pkg/core"
)

func getRunnerOptions() *core.RunnerOptions {
	r := core.RunnerOptions{
		ContainerName:  "test-container",
		ContainerArgs:  []string{"sleep", "10"},
		DockerImage:    "ubuntu:latest",
		HostVolumePath: "/tmp",
		PodType:        core.ParsingPod,
	}
	return &r
}

func TestDockerDestroy(t *testing.T) {
	ctx := context.Background()
	runnerOpts := getRunnerOptions()
	// test create container
	statusCreate := runner.Create(ctx, runnerOpts)
	if !statusCreate.Done {
		t.Errorf("error creating container: %v", statusCreate.Error)
	}
	if err := runner.Destroy(ctx, runnerOpts); err != nil {
		t.Errorf("error destroying container: %v", err)
	}
}
