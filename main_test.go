package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	"github.com/zenizh/go-capturer"
)

func TestDefault(t *testing.T) {
	cwd, err := os.Getwd()
	assert.Nil(t, err)
	os.Setenv("AWS_CONFIG_FILE", filepath.Join(cwd, "test", "fixtures", "aws_config"))
	defer os.Unsetenv("AWS_CONFIG_FILE")
	os.Setenv("AWS_PROFILE", "fake")
	defer os.Unsetenv("AWS_PROFILE")

	stdout := capturer.CaptureStdout(func() {
		main()
	})

	// Disable Loop Detection
	os.Setenv("_AWS_CRED_CACHIER_CSUM", "TEST")
	defer os.Unsetenv("_AWS_CRED_CACHIER_CSUM")

	stderr := capturer.CaptureStderr(func() {
		// Mask STDOUT
		os.Stdout = os.NewFile(0, os.DevNull)
		main()
	})

	var cred AwsProcessCredential

	err = json.Unmarshal([]byte(stdout), &cred)
	assert.Nil(t, err, "STDOUT should be a valid JSON AwsProcessCredential")

	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", cred.AccessKeyID)
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", cred.SecretAccessKey)

	assert.Empty(t, stderr, "STDERR should be empty")
}

func TestLoop(t *testing.T) {
	cwd, err := os.Getwd()
	assert.Nil(t, err)
	os.Setenv("AWS_CONFIG_FILE", filepath.Join(cwd, "test", "fixtures", "aws_config"))
	defer os.Unsetenv("AWS_CONFIG_FILE")
	os.Setenv("AWS_PROFILE", "fake")
	defer os.Unsetenv("AWS_PROFILE")

	fakeLogFatal := func(msg ...interface{}) {
		assert.Equal(t, "Loop detected! Called recursively by PID: ", msg[0])
		panic("log.Fatal called")
	}

	patch := monkey.Patch(log.Fatal, fakeLogFatal)
	defer patch.Unpatch()

	// First Invocation
	main()
	// Second Invocation
	assert.PanicsWithValue(t, "log.Fatal called", main, "log.Fatal was not called")
}
