package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/require"

	"github.com/fornellas/resonance/internal/resource"
)

func CreateResourceYamls(t *testing.T, resourcesRoot string, resourceDefsMap map[string]resource.ResourceDefs) {
	require.NoError(t, os.Mkdir(resourcesRoot, os.FileMode(0700)))
	for name, resources := range resourceDefsMap {
		bundleBytes, err := yaml.Marshal(resources)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(resourcesRoot, name), bundleBytes, os.FileMode(0600),
		); err != nil {
			t.Fatal(err)
		}
	}
}

func captureOutput(t *testing.T, fn func()) (stdout string, stderr string) {
	originalStdout := os.Stdout
	readStdout, writeStdout, _ := os.Pipe()
	os.Stdout = writeStdout
	defer func() { os.Stdout = originalStdout }()
	stdoutBuf := bytes.NewBuffer([]byte{})
	multiWriterStdout := io.MultiWriter(originalStdout, stdoutBuf)
	stdoutCh := make(chan bool, 1)
	go func() {
		if _, err := io.Copy(multiWriterStdout, readStdout); err != nil {
			t.Errorf("failed to copy Stdout: %s", err)
		}
		stdoutCh <- true
	}()

	originalStderr := os.Stderr
	readStderr, writeStderr, _ := os.Pipe()
	os.Stderr = writeStderr
	defer func() { os.Stderr = originalStderr }()
	stderrBuf := bytes.NewBuffer([]byte{})
	multiWriterStderr := io.MultiWriter(originalStderr, stderrBuf)
	stderrCh := make(chan bool)
	go func() {
		if _, err := io.Copy(multiWriterStderr, readStderr); err != nil {
			t.Errorf("failed to copy Stderr: %s", err)
		}
		stderrCh <- true
	}()

	func() {
		fn()

		defer func() {
			writeStdout.Close()
			<-stdoutCh
			readStdout.Close()

			writeStderr.Close()
			<-stderrCh
			readStderr.Close()
		}()
	}()

	return stdoutBuf.String(), stderrBuf.String()
}

type TestCmd struct {
	Args                 []string
	ExpectedCode         int
	ExpectStdoutContains string
	ExpectStderrContains string
}

func (c TestCmd) String() string {
	return strings.Join(append([]string{"resonance"}, c.Args...), " ")
}

func (c *TestCmd) Run(t *testing.T) {
	osExitErr := errors.New("os.Exit")
	originalExit := Exit
	t.Cleanup(func() { Exit = originalExit })
	Exit = func(code int) {
		if c.ExpectedCode != code {
			t.Fatalf("%v exited %d, expected %d", c, code, c.ExpectedCode)
		}
		panic(osExitErr)
	}
	defer func() {
		if r := recover(); r != nil && r != osExitErr {
			panic(r)
		}
	}()

	t.Cleanup(func() { ResetFlags() })

	RootCmd.SetArgs(c.Args)

	stdout, stderr := captureOutput(t, func() {
		if err := RootCmd.Execute(); err != nil {
			t.Fatal(err)
		}
	})

	if c.ExpectStdoutContains != "" {
		require.Contains(t, stdout, c.ExpectStdoutContains, "stdout does not contain expected content")
	}

	if c.ExpectStderrContains != "" {
		require.Contains(t, stderr, c.ExpectStderrContains, "stderr does not contain expected content")
	}
}
