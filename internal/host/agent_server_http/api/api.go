package api

import (
	"errors"
	"os"
	"os/user"
	"strings"

	"github.com/fornellas/resonance/host"
)

type FileAction int

const (
	Chmod FileAction = iota
	Chown
	Mkdir
)

type File struct {
	Action FileAction
	Mode   uint32
	Uid    uint32
	Gid    uint32
}

type Error struct {
	Type    string
	Message string
}

func (e Error) Error() error {
	switch e.Type {
	case "ErrPermission":
		return os.ErrPermission
	case "ErrNotExist":
		return os.ErrNotExist
	case "UnknownUserError":
		return user.UnknownUserError(strings.TrimPrefix(e.Message, user.UnknownUserError("").Error()))
	case "UnknownGroupError":
		return user.UnknownGroupError(strings.TrimPrefix(e.Message, user.UnknownGroupError("").Error()))
	case "ErrExist":
		return os.ErrExist
	default:
		return errors.New(e.Message)
	}
}

type Cmd struct {
	Path   string
	Args   []string
	Env    []string
	Dir    string
	Stdin  []byte
	Stdout bool
	Stderr bool
}

type CmdResponse struct {
	WaitStatus host.WaitStatus
	Stdout     []byte
	Stderr     []byte
}
