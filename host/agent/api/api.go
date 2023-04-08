package api

import (
	"errors"
	"os"
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
	default:
		return errors.New(e.Message)
	}
}
