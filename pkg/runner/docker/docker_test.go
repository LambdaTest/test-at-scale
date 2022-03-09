package docker

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/google/uuid"
)

func getRunnerOptions() *core.RunnerOptions {
	os.Setenv(global.AutoRemoveEnv, strconv.FormatBool(true))
	r := core.RunnerOptions{
		ContainerName:  fmt.Sprintf("test-container-%s", uuid.NewString()),
		ContainerArgs:  []string{"sleep", "10"},
		DockerImage:    "ubuntu:latest",
		HostVolumePath: "/tmp",
		PodType:        core.ParsingPod,
	}
	return &r
}

func TestDockerCreate(t *testing.T) {
	ctx := context.Background()
	runnerOpts := getRunnerOptions()
	// test create container
	statusCreate := runner.Create(ctx, runnerOpts)
	if !statusCreate.Done {
		t.Errorf("error creating container: %v", statusCreate.Error)
	}

}

func TestDockerRun(t *testing.T) {
	ctx := context.Background()
	runnerOpts := getRunnerOptions()
	// test create container
	statusCreate := runner.Create(ctx, runnerOpts)
	if !statusCreate.Done {
		t.Errorf("error creating container: %v", statusCreate.Error)
	}
	if status := runner.Run(ctx, runnerOpts); !status.Done {
		t.Errorf("error in running container : %v", status.Error)
		return
	}

}

func TestDockerWaitCompletion(t *testing.T) {
	ctx := context.Background()
	runnerOpts := getRunnerOptions()
	// test create container
	statusCreate := runner.Create(ctx, runnerOpts)
	if !statusCreate.Done {
		t.Errorf("error creating container: %v", statusCreate.Error)
	}
	if status := runner.Run(ctx, runnerOpts); !status.Done {
		t.Errorf("error in running container : %v", status.Error)
		return
	}
	if err := runner.WaitForCompletion(ctx, runnerOpts); err != nil {
		t.Errorf("Error while waiting for completion of container")
	}

}

func TestDockerDestroyWithoutRunning(t *testing.T) {
	ctx := context.Background()
	os.Setenv(global.AutoRemoveEnv, strconv.FormatBool(false))
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

func TestDockerDestroyWithRunningWoAutoRemove(t *testing.T) {
	ctx := context.Background()
	runnerOpts := getRunnerOptions()
	// test create container
	os.Setenv(global.AutoRemoveEnv, strconv.FormatBool(false))
	statusCreate := runner.Create(ctx, runnerOpts)
	if !statusCreate.Done {
		t.Errorf("error creating container: %v", statusCreate.Error)
	}
	if status := runner.Run(ctx, runnerOpts); !status.Done {
		t.Errorf("error in running container : %v", status.Error)
		return
	}
	if err := runner.Destroy(ctx, runnerOpts); err != nil {
		t.Errorf("error destroying container: %v", err)
	}
}

func TestDockerDestroyWithRunningWithAutoRemove(t *testing.T) {
	ctx := context.Background()
	runnerOpts := getRunnerOptions()
	// test create container
	os.Setenv(global.AutoRemoveEnv, strconv.FormatBool(true))
	statusCreate := runner.Create(ctx, runnerOpts)
	if !statusCreate.Done {
		t.Errorf("error creating container: %v", statusCreate.Error)
	}
	if status := runner.Run(ctx, runnerOpts); !status.Done {
		t.Errorf("error in running container : %v", status.Error)
		return
	}
	if err := runner.Destroy(ctx, runnerOpts); err != nil {
		t.Errorf("error destroying container: %v", err)
	}
}

func TestDockerPullAlways(t *testing.T) {
	runnerOpts := getRunnerOptions()
	// test create container
	runnerOpts.PodType = core.NucleusPod
	if err := runner.PullImage(&core.ContainerImageConfig{
		Mode:       config.PublicMode,
		PullPolicy: config.PullAlways,
		Image:      runnerOpts.DockerImage,
	}, runnerOpts); err != nil {
		t.Errorf("Error while pulling image %v", err)
	}
}

func TestDockerPullNever(t *testing.T) {
	runnerOpts := getRunnerOptions()
	// test create container
	runnerOpts.PodType = core.NucleusPod
	if err := runner.PullImage(&core.ContainerImageConfig{
		Mode:       config.PublicMode,
		PullPolicy: config.PullNever,
		Image:      "dummy-image",
	}, runnerOpts); err != nil {
		t.Errorf("Error while pulling image %v", err)
	}
}
