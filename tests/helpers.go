package tests

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/cli"
	"github.com/fornellas/resonance/resource"
)

func setupTestType(t *testing.T, testFuncCalls []TestFuncCall) {
	TestT = t
	TestExpectedFuncCalls = testFuncCalls
	t.Cleanup(func() {
		if len(TestExpectedFuncCalls) > 0 {
			t.Errorf("expected calls pending:\n%v", TestExpectedFuncCalls)
		}
	})
}

func setupDirs(t *testing.T) (string, string) {
	prefix := t.TempDir()

	stateRoot := filepath.Join(prefix, "state")
	if err := os.Mkdir(stateRoot, os.FileMode(0700)); err != nil {
		t.Fatal(err)
	}

	resourcesRoot := filepath.Join(prefix, "resources")
	if err := os.Mkdir(resourcesRoot, os.FileMode(0700)); err != nil {
		t.Fatal(err)
	}

	return stateRoot, resourcesRoot
}

func setupBundles(t *testing.T, resourcesRoot string, resourcesMap map[string]resource.Resources) {
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

type Cmd struct {
	Args             []string
	ExpectedCode     int
	ExpectedInOutput string
	ExpectedInStdout string
}

func (c Cmd) String() string {
	return strings.Join(c.Args, " ")
}

func removeANSIEscapeSequences(input string) string {
	ansiEscapeRegex := regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	return ansiEscapeRegex.ReplaceAllString(input, "")
}

func runCommand(t *testing.T, cmd Cmd) {
	var outputBuffer bytes.Buffer

	// log output function
	cmdOutputLogged := false
	logCmdOutput := func() {
		if !cmdOutputLogged && (!testing.Verbose() || t.Failed()) {
			t.Logf("%s\n%s", cmd, outputBuffer.String())
		}
		cmdOutputLogged = true
	}
	t.Cleanup(func() { logCmdOutput() })
	type expectedExit struct{}

	// Exit mock
	cli.ExitFunc = func(code int) {
		if cmd.ExpectedCode != code {
			logCmdOutput()
			t.Fatalf("expected exit code %d: got %d", cmd.ExpectedCode, code)
		}
		panic(expectedExit{})
	}
	defer func() {
		switch p := recover(); p {
		case nil:
			if cmd.ExpectedCode != 0 {
				logCmdOutput()
				t.Fatalf("expected exit code %d: got %d", cmd.ExpectedCode, 0)
			}
		case expectedExit{}:
		default:
			logCmdOutput()
			panic(p)
		}
		if cmd.ExpectedInOutput != "" {
			if !strings.Contains(removeANSIEscapeSequences(outputBuffer.String()), cmd.ExpectedInOutput) {
				logCmdOutput()
				t.Fatalf("output does not contain %#v", cmd.ExpectedInOutput)
			}
		}
	}()

	command := cli.Cmd

	command.SetArgs(cmd.Args)

	// Capture output
	var output io.Writer
	if testing.Verbose() {
		output = io.MultiWriter(&outputBuffer, os.Stdout)
		t.Logf("%s", cmd)
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
		if cmd.ExpectedInStdout != "" {
			if !strings.Contains(removeANSIEscapeSequences(stdoutStr), cmd.ExpectedInStdout) {
				logCmdOutput()
				t.Logf("stdout:\n%s", stdoutStr)
				t.Fatalf("stdout does not contain %#v", cmd.ExpectedInStdout)
			}
		}
	}()

	cli.Reset()

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
}
