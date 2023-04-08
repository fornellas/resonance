package api

import (
	"errors"
	"os"
	"os/user"
	"strings"
	// "gopkg.in/yaml.v3"
)

// Action to be executed for a resource.
type FileAction int

const (
	Chmod FileAction = iota
	Chown
	Mkdir
)

type File struct {
	Action FileAction
	Mode   os.FileMode
	Uid    int
	Gid    int
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
