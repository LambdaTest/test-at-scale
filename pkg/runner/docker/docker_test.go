package docker

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/synapse"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func getRunnerOptions() *core.RunnerOptions {
	os.Setenv(global.AutoRemoveEnv, strconv.FormatBool(true))

	containerName := fmt.Sprintf("test-container-%s", uuid.NewString())
	r := core.RunnerOptions{
		ContainerName:  containerName,
		ContainerArgs:  []string{"sleep", "10"},
		DockerImage:    "alpine:latest",
		HostVolumePath: "/tmp",
		PodType:        core.NucleusPod,
		Label:          map[string]string{synapse.BuildID: containerName},
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

func TestDockerVolumes(t *testing.T) {
	ctx := context.Background()
	runnerOpts := getRunnerOptions()

	os.Setenv(global.AutoRemoveEnv, strconv.FormatBool(true))
	statusCreate := runner.Create(ctx, runnerOpts)
	if !statusCreate.Done {
		t.Errorf("error creating container: %v", statusCreate.Error)
	}

	correctVolumeName := fmt.Sprintf("%s-%s", volumePrefix, runnerOpts.Label[synapse.BuildID])
	incorrectVolumeName := fmt.Sprintf("incorrect-%s-%s", volumePrefix, runnerOpts.Label[synapse.BuildID])

	exists, err := runner.FindVolumes(incorrectVolumeName)
	if err != nil {
		t.Errorf("error finding docker volume: %v", err)
	}
	assert.Equal(t, false, exists)

	exists, err = runner.FindVolumes(correctVolumeName)
	if err != nil {
		t.Errorf("error finding docker volume: %v", err)
	}
	assert.Equal(t, true, exists)

	if status := runner.Run(ctx, runnerOpts); !status.Done {
		t.Errorf("error in running container : %v", status.Error)
		return
	}

	expectedFileContent := `{"access_token":"dummytoken","expiry":"0001-01-01T00:00:00Z","refresh_token":"","token_type":"Bearer"}`
	secretBytes, err := secretsManager.GetGitSecretBytes()
	if err != nil {
		t.Errorf("error retrieving secrets: %v", err)
	}
	assert.Equal(t, expectedFileContent, string(secretBytes))

	if err = runner.Destroy(ctx, runnerOpts); err != nil {
		t.Errorf("error destroying container: %v", err)
	}
}
