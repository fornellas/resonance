package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
			require.NoError(t, writeStdout.Close())
			<-stdoutCh
			require.NoError(t, readStdout.Close())

			require.NoError(t, writeStderr.Close())
			<-stderrCh
			require.NoError(t, readStderr.Close())
		}()
	}()

	return stdoutBuf.String(), stderrBuf.String()
}

type TestCmd struct {
	Args                 []string
	ExpectedCode         int
	ExpectStdoutContains []string
	ExpectStderrContains []string
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

	for _, str := range c.ExpectStdoutContains {
		require.Contains(t, stdout, str, "stdout does not contain expected content")
	}

	for _, str := range c.ExpectStderrContains {
		require.Contains(t, stderr, str, "stderr does not contain expected content")
	}
}
