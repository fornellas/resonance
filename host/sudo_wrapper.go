package host

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"al.essio.dev/pkg/shellescape"
	"golang.org/x/term"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

// StdinSudo prevents stdin from being read, before we can detect output
// from sudo on stdout. This is required because os/exec and ssh buffer stdin
// before there's any read, meaning we can't intercept the sudo prompt
// reliably
type StdinSudo struct {
	Unlock   chan struct{}
	SendPass chan string
	Reader   io.Reader
	mutex    sync.Mutex
	unlocked bool
}

func (r *StdinSudo) Read(p []byte) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.unlocked {
		select {
		case <-r.Unlock:
			r.unlocked = true
		case password := <-r.SendPass:
			passwordBytes := []byte(fmt.Sprintf("%s\n", password))
			if len(passwordBytes) > len(p) {
				return 0, fmt.Errorf(
					"password is longer (%d) than read buffer (%d)", len(passwordBytes), len(p),
				)
			}
			copy(p, passwordBytes)
			return len(passwordBytes), nil
		}
	}

	return r.Reader.Read(p)
}

// StderrSudo waits for either write:
// - sudo prompt: asks for password, caches it, and send to stdin.
// - sudo ok: unlocks stdin.
type StderrSudo struct {
	Unlock          chan struct{}
	SendPass        chan string
	Prompt          []byte
	SudoOk          []byte
	Writer          io.Writer
	Password        **string
	mutex           sync.Mutex
	unlocked        bool
	passwordAttempt *string
}

func (w *StderrSudo) Write(p []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	var extraLen int

	if !w.unlocked {
		if bytes.Contains(p, w.Prompt) {
			var password string
			if *w.Password == nil {
				state, err := term.MakeRaw(int(os.Stdin.Fd()))
				if err != nil {
					return 0, err
				}
				defer term.Restore(int(os.Stdin.Fd()), state)

				var passwordBytes []byte
				fmt.Printf("sudo password: ")
				passwordBytes, err = (term.ReadPassword(int(os.Stdin.Fd())))
				if err != nil {
					return 0, err
				}
				fmt.Printf("\n\r")
				password = string(passwordBytes)
				w.passwordAttempt = &password
			} else {
				password = **w.Password
			}
			w.SendPass <- password
			extraLen = len(w.Prompt)
			p = bytes.ReplaceAll(p, w.Prompt, []byte{})
		} else if bytes.Contains(p, w.SudoOk) {
			if w.passwordAttempt != nil {
				*w.Password = w.passwordAttempt
			}
			w.Unlock <- struct{}{}
			w.unlocked = true
			extraLen = len(w.SudoOk)
			p = bytes.ReplaceAll(p, w.SudoOk, []byte{})
		}
	}

	len, err := w.Writer.Write(p)
	return len + extraLen, err
}

// SudoWrapper wraps another BaseHost and runs all commands with sudo.
type SudoWrapper struct {
	BaseHost types.BaseHost
	Password *string
	envPath  string
}

func NewSudoWrapper(ctx context.Context, hst types.BaseHost) (*SudoWrapper, error) {
	ctx, _ = log.MustContextLoggerWithSection(ctx, "⚡ Sudo")

	sudoWrapper := SudoWrapper{
		BaseHost: hst,
	}

	cmd := types.Cmd{
		Path: "true",
	}
	waitStatus, err := sudoWrapper.Run(ctx, cmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		return nil, fmt.Errorf("failed to run %s: %s", cmd, waitStatus.String())
	}

	if err := sudoWrapper.setEnvPath(ctx); err != nil {
		return nil, err
	}

	return &sudoWrapper, nil
}

func (h *SudoWrapper) getRandomString() string {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])
}

func (h *SudoWrapper) runEnv(ctx context.Context, cmd types.Cmd, ignoreCmdEnv bool) (types.WaitStatus, error) {
	prompt := fmt.Sprintf("sudo password (%s)", h.getRandomString())
	sudoOk := fmt.Sprintf("sudo ok (%s)", h.getRandomString())

	shellCmdArgs := []string{shellescape.Quote(cmd.Path)}
	for _, arg := range cmd.Args {
		shellCmdArgs = append(shellCmdArgs, shellescape.Quote(arg))
	}
	shellCmdStr := strings.Join(shellCmdArgs, " ")

	if cmd.Dir == "" {
		cmd.Dir = "/tmp"
	}

	cmd.Path = "sudo"

	if !ignoreCmdEnv {
		if len(cmd.Env) == 0 {
			cmd.Env = []string{"LANG=en_US.UTF-8"}
			if h.envPath != "" {
				cmd.Env = append(cmd.Env, h.envPath)
			}
		}
		envStrs := []string{}
		for _, nameValue := range cmd.Env {
			envStrs = append(envStrs, shellescape.Quote(nameValue))
		}
		cmd.Args = []string{
			"--stdin",
			"--prompt", prompt,
			"--", "sh", "-c",
			fmt.Sprintf(
				"echo -n %s 1>&2 && cd %s && exec env --ignore-environment %s %s",
				shellescape.Quote(sudoOk), cmd.Dir, strings.Join(envStrs, " "), shellCmdStr,
			),
		}
	} else {
		cmd.Args = []string{
			"--stdin",
			"--prompt", prompt,
			"--", "sh", "-c",
			fmt.Sprintf(
				"echo -n %s 1>&2 && cd %s && exec %s",
				shellescape.Quote(sudoOk), cmd.Dir, shellCmdStr,
			),
		}
	}

	unlockStdin := make(chan struct{}, 1)
	sendPassStdin := make(chan string, 1)

	var stdin io.Reader
	if cmd.Stdin != nil {
		stdin = cmd.Stdin
	} else {
		stdin = &bytes.Buffer{}
	}
	cmd.Stdin = &StdinSudo{
		Unlock:   unlockStdin,
		SendPass: sendPassStdin,
		Reader:   stdin,
	}

	var stderr io.Writer
	if cmd.Stderr != nil {
		stderr = cmd.Stderr
	} else {
		stderr = io.Discard
	}
	cmd.Stderr = &StderrSudo{
		Unlock:   unlockStdin,
		SendPass: sendPassStdin,
		Prompt:   []byte(prompt),
		SudoOk:   []byte(sudoOk),
		Writer:   stderr,
		Password: &h.Password,
	}

	return h.BaseHost.Run(ctx, cmd)
}

func (h *SudoWrapper) Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
	return h.runEnv(ctx, cmd, false)
}

func (h SudoWrapper) String() string {
	return h.BaseHost.String()
}

func (h SudoWrapper) Type() string {
	return h.BaseHost.Type()
}

func (h SudoWrapper) Close(ctx context.Context) error {
	return h.BaseHost.Close(ctx)
}

func (h *SudoWrapper) setEnvPath(ctx context.Context) error {
	stdoutBuffer := bytes.Buffer{}
	stderrBuffer := bytes.Buffer{}
	cmd := types.Cmd{
		Path:   "env",
		Stdout: &stdoutBuffer,
		Stderr: &stderrBuffer,
	}
	waitStatus, err := h.runEnv(ctx, cmd, true)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		return fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdoutBuffer.String(), stderrBuffer.String(),
		)
	}
	for _, value := range strings.Split(stdoutBuffer.String(), "\n") {
		if strings.HasPrefix(value, "PATH=") {
			h.envPath = value
			break
		}
	}
	return nil
}
