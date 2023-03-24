package host

import (
	"context"
	"fmt"
	"os"

	"github.com/fornellas/resonance/log"
)

// Sudo implements Host interface by having all methods rely on an underlying Host.Run, and
// preceding all commands with sudo.
type Sudo struct {
	baseRun
	Host Host
	// password *string
}

// func getRandomPrompt() string {
// 	bytes := make([]byte, 64)
// 	_, err := rand.Read(bytes)
// 	if err != nil {
// 		panic(err)
// 	}
// 	hash := sha512.Sum512(bytes)
// 	return hex.EncodeToString(hash[:])
// }

func (s Sudo) Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {
	// prompt := getRandomPrompt()
	prompt := "sudo password: "
	cmd.Args = append([]string{"--stdin", "--prompt", prompt, "--", cmd.Path}, cmd.Args...)
	cmd.Path = "sudo"

	// stderr
	// if first bytes == prompt
	//   if s.password == nil
	//     s.password = readPassword()
	//   fmt.Fprintf(stdin, "%s\n", s.password)

	return s.Host.Run(ctx, cmd)
}

func (s Sudo) String() string {
	return s.Host.String()
}

func (s Sudo) Close() error {
	return s.Host.Close()
}

func NewSudo(ctx context.Context, host Host) (Sudo, error) {
	logger := log.GetLogger(ctx)
	logger.Info("⚡ Sudo access")
	nestedCtx := log.IndentLogger(ctx)

	sudoHost := Sudo{
		Host: host,
	}
	sudoHost.baseRun.Host = sudoHost

	cmd := Cmd{
		Path:  "true",
		Stdin: os.Stdin,
	}
	waitStatus, err := sudoHost.Run(nestedCtx, cmd)
	if err != nil {
		return Sudo{}, err
	}
	if !waitStatus.Success() {
		return Sudo{}, fmt.Errorf("failed to run %s: %s", cmd, waitStatus.String())
	}

	return sudoHost, nil
}
