package host

import (
	"context"
	"fmt"
	"os"
	"os/user"
)

// Agent interacts with a given Host using an agent that's copied and ran at the
// host.
type Agent struct {
	Host Host
}

func (a Agent) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	return fmt.Errorf("TODO Agent.Chmod")
}

func (a Agent) Chown(ctx context.Context, name string, uid, gid int) error {
	return fmt.Errorf("TODO Agent.Chown")
}

func (a Agent) Lookup(ctx context.Context, username string) (*user.User, error) {
	return nil, fmt.Errorf("TODO Agent.Lookup")
}

func (a Agent) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	return nil, fmt.Errorf("TODO Agent.LookupGroup")
}

func (a Agent) Lstat(ctx context.Context, name string) (HostFileInfo, error) {
	return HostFileInfo{}, fmt.Errorf("TODO Agent.Lstat")
}

func (a Agent) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return fmt.Errorf("TODO Agent.Mkdir")
}

func (a Agent) ReadFile(ctx context.Context, name string) ([]byte, error) {
	return nil, fmt.Errorf("TODO Agent.ReadFile")
}

func (a Agent) Remove(ctx context.Context, name string) error {
	return fmt.Errorf("TODO Agent.Remove")
}

func (a Agent) Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {
	return WaitStatus{}, fmt.Errorf("TODO Agent.Run")
}

func (a Agent) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	return fmt.Errorf("TODO Agent.WriteFile")
}

func (a Agent) String() string {
	return a.Host.String()
}

func (a Agent) Close() error {
	// TODO rm agent
	return a.Host.Close()
}

func NewAgent(ctx context.Context, host Host) (Agent, error) {
	agent := Agent{
		Host: host,
	}

	// TODO get host arch
	// TODO copy agent to host
	// TODO spawn agent
	// TODO create client

	return agent, nil
}
