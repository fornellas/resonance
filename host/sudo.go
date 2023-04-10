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

	"github.com/alessio/shellescape"
	"golang.org/x/term"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

// Sudo implements Host interface by having all methods rely on an underlying Host.Run, and
// preceding all commands with sudo.
type Sudo struct {
	baseRun
	Host     Host
	Password *string
	envPath  string
}

// stdinSudo prevents stdin from being read, before we can detect output
// from sudo on stdout. This is required because os/exec and ssh buffer stdin
// before there's any read, meaning we can't intercept the sudo prompt
// reliably
type stdinSudo struct {
	Unlock   chan struct{}
	SendPass chan string
	Reader   io.Reader
	mutex    sync.Mutex
	unlocked bool
}

func (sis *stdinSudo) Read(p []byte) (int, error) {
	sis.mutex.Lock()
	defer sis.mutex.Unlock()

	if !sis.unlocked {
		select {
		case <-sis.Unlock:
			sis.unlocked = true
		case password := <-sis.SendPass:
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

	return sis.Reader.Read(p)
}

// stderrSudo waits for either write:
// - sudo prompt: asks for password, caches it, and send to stdin.
// - sudo ok: unlocks stdin.
type stderrSudo struct {
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

func (ses *stderrSudo) Write(p []byte) (int, error) {
	ses.mutex.Lock()
	defer ses.mutex.Unlock()

	var extraLen int

	if !ses.unlocked {
		if bytes.Contains(p, ses.Prompt) {
			var password string
			if *ses.Password == nil {
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
				ses.passwordAttempt = &password
			} else {
				password = **ses.Password
			}
			ses.SendPass <- password
			extraLen = len(ses.Prompt)
			p = bytes.ReplaceAll(p, ses.Prompt, []byte{})
		} else if bytes.Contains(p, ses.SudoOk) {
			if ses.passwordAttempt != nil {
				*ses.Password = ses.passwordAttempt
			}
			ses.Unlock <- struct{}{}
			ses.unlocked = true
			extraLen = len(ses.SudoOk)
			p = bytes.ReplaceAll(p, ses.SudoOk, []byte{})
		}
	}

	len, err := ses.Writer.Write(p)
	return len + extraLen, err
}

func getRandomString() string {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])
}

func (s *Sudo) runEnv(ctx context.Context, cmd types.Cmd, ignoreCmdEnv bool) (types.WaitStatus, error) {
	prompt := fmt.Sprintf("sudo password (%s)", getRandomString())
	sudoOk := fmt.Sprintf("sudo ok (%s)", getRandomString())

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
			if s.envPath != "" {
				cmd.Env = append(cmd.Env, s.envPath)
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
	cmd.Stdin = &stdinSudo{
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
	cmd.Stderr = &stderrSudo{
		Unlock:   unlockStdin,
		SendPass: sendPassStdin,
		Prompt:   []byte(prompt),
		SudoOk:   []byte(sudoOk),
		Writer:   stderr,
		Password: &s.Password,
	}

	return s.Host.Run(ctx, cmd)
}

func (s *Sudo) Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, error) {
	return s.runEnv(ctx, cmd, false)
}

func (s Sudo) String() string {
	return s.Host.String()
}

func (s Sudo) Close() error {
	return s.Host.Close()
}

func (s *Sudo) setEnvPath(ctx context.Context) error {
	stdoutBuffer := bytes.Buffer{}
	stderrBuffer := bytes.Buffer{}
	cmd := types.Cmd{
		Path:   "env",
		Stdout: &stdoutBuffer,
		Stderr: &stderrBuffer,
	}
	waitStatus, err := s.runEnv(ctx, cmd, true)
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
			s.envPath = value
			break
		}
	}
	return nil
}

func NewSudo(ctx context.Context, host Host) (*Sudo, error) {
	logger := log.GetLogger(ctx)
	logger.Info("⚡ Sudo")
	nestedCtx := log.IndentLogger(ctx)

	sudoHost := Sudo{
		Host: host,
	}
	sudoHost.baseRun.Host = &sudoHost

	cmd := types.Cmd{
		Path: "true",
	}
	waitStatus, err := sudoHost.Run(nestedCtx, cmd)
	if err != nil {
		return nil, err
	}
	if !waitStatus.Success() {
		return nil, fmt.Errorf("failed to run %s: %s", cmd, waitStatus.String())
	}

	if err := sudoHost.setEnvPath(nestedCtx); err != nil {
		return nil, err
	}

	return &sudoHost, nil
}
