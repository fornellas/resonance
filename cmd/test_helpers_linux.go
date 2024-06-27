package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/internal/resource"
)

func SetupBundles(t *testing.T, resourcesRoot string, resourcesMap map[string]resource.Resources) {
	for name, resources := range resourcesMap {
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

type TestCmd struct {
	Args             []string
	ExpectedCode     int
	ExpectedInOutput string
	ExpectedInStdout string
}

func (c TestCmd) String() string {
	return strings.Join(c.Args, " ")
}

func (c *TestCmd) Run(t *testing.T) {
	var outputBuffer bytes.Buffer

	// log output function
	cmdOutputLogged := false
	logCmdOutput := func() {
		if !cmdOutputLogged && (!testing.Verbose() && t.Failed()) {
			t.Logf("%s\n%s", c, outputBuffer.String())
		}
		cmdOutputLogged = true
	}
	t.Cleanup(func() { logCmdOutput() })
	type expectedExit struct{}

	// Exit mock
	ExitFunc = func(code int) {
		if c.ExpectedCode != code {
			logCmdOutput()
			t.Fatalf("expected exit code %d: got %d", c.ExpectedCode, code)
		}
		panic(expectedExit{})
	}
	defer func() {
		switch p := recover(); p {
		case nil:
			if c.ExpectedCode != 0 {
				logCmdOutput()
				t.Fatalf("expected exit code %d: got %d", c.ExpectedCode, 0)
			}
		case expectedExit{}:
		default:
			logCmdOutput()
			panic(p)
		}
		if c.ExpectedInOutput != "" {
			if !strings.Contains(outputBuffer.String(), c.ExpectedInOutput) {
				logCmdOutput()
				t.Fatalf("output does not contain %#v", c.ExpectedInOutput)
			}
		}
	}()

	command := RootCmd

	command.SetArgs(c.Args)

	// Capture output
	var output io.Writer
	if testing.Verbose() {
		output = io.MultiWriter(&outputBuffer, os.Stdout)
		t.Logf("%s", c)
	} else {
		output = &outputBuffer
	}
	command.SetOut(output)

	// Capture stdout
	// TODO print stdout if testing.Verbose()
	originalStdout := os.Stdout
	stdoutRead, stdoutWrite, _ := os.Pipe()
	os.Stdout = stdoutWrite
	stdoutCh := make(chan string)
	go func() {
		var buff bytes.Buffer
		_, err := io.Copy(&buff, stdoutRead)
		if err != nil {
			t.Errorf("failed to read stdout: %s", err)
		}
		stdoutCh <- buff.String()
	}()
	defer func() {
		os.Stdout = originalStdout
		stdoutWrite.Close()
		stdoutStr := <-stdoutCh
		if c.ExpectedInStdout != "" {
			if !strings.Contains(stdoutStr, c.ExpectedInStdout) {
				logCmdOutput()
				t.Logf("stdout:\n%s", stdoutStr)
				t.Fatalf("stdout does not contain %#v", c.ExpectedInStdout)
			}
		}
	}()

	Reset()

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
}
