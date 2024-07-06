package host

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"testing"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

type FnChmod struct {
	Name        string
	FileModeode os.FileMode
	ReturnError error
}

type FnChown struct {
	Name        string
	Uid         int
	Gid         int
	ReturnError error
}

type FnLookup struct {
	Username    string
	ReturnUser  *user.User
	ReturnError error
}

type FnLookupGroup struct {
	Name        string
	ReturnGroup *user.Group
	ReturnError error
}

type FnLstat struct {
	Name           string
	ReturnFileInfo os.FileInfo
	ReturnError    error
}

type FnReadFile struct {
	Name        string
	ReturnBytes []byte
	ReturnError error
}

type FnRemove struct {
	Name        string
	ReturnError error
}

type FnRun struct {
	Cmd              host.Cmd
	ReturnWaitStatus host.WaitStatus
	ReturnStdout     string
	ReturnStderr     string
	ReturnError      error
}

type FnWriteFile struct {
	Name        string
	Data        []byte
	Perm        os.FileMode
	ReturnError error
}

type FnCall struct {
	Chmod       *FnChmod
	Chown       *FnChown
	Lookup      *FnLookup
	LookupGroup *FnLookupGroup
	Lstat       *FnLstat
	ReadFile    *FnReadFile
	Remove      *FnRemove
	Run         *FnRun
	WriteFile   *FnWriteFile
}

// TestHost aids testing by enabling mocking of host functions.
type TestHost struct {
	T               *testing.T
	ExpectedFnCalls []FnCall
}

func (t *TestHost) getFnCall() *FnCall {
	if len(t.ExpectedFnCalls) == 0 {
		return nil
	}
	testFnCall, expectedFnCalls := t.ExpectedFnCalls[0], t.ExpectedFnCalls[1:]
	t.ExpectedFnCalls = expectedFnCalls
	return &testFnCall
}

func (t TestHost) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	logger := log.MustLoggerIndented(ctx)
	logger.Debug("TestHost.Chmod", "name", name, "mode", mode)
	funcCall := t.getFnCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: Chmod(%v, %v)", name, mode)
	}
	if funcCall.Chmod == nil {
		t.T.Fatalf("unexpected call: got Chmod(%v, %v), expected %#v", name, mode, funcCall)
	}
	return funcCall.Chmod.ReturnError
}

func (t TestHost) Chown(ctx context.Context, name string, uid, gid int) error {
	logger := log.MustLoggerIndented(ctx)
	logger.Debug("TestHost.Chown", "name", name, "uid", uid, "gid", gid)
	funcCall := t.getFnCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: Chown(%v, %v, %v)", name, uid, gid)
	}
	if funcCall.Chown == nil {
		t.T.Fatalf("unexpected call: got Chown(%v, %v, %v), expected %#v", name, uid, gid, funcCall)
	}
	return funcCall.Chown.ReturnError
}

func (t TestHost) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.MustLoggerIndented(ctx)
	logger.Debug("TestHost.Lookup", "username", username)
	funcCall := t.getFnCall()
	if funcCall == nil {
		return nil, fmt.Errorf("no more calls expected: got Lookup(%v)", username)
	}
	if funcCall.Lookup == nil {
		t.T.Fatalf("unexpected call: got Lookup(%v), expected %#v", username, funcCall)
	}
	return funcCall.Lookup.ReturnUser, funcCall.Lookup.ReturnError
}

func (t TestHost) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.MustLoggerIndented(ctx)
	logger.Debug("TestHost.LookupGroup", "name", name)
	funcCall := t.getFnCall()
	if funcCall == nil {
		return nil, fmt.Errorf("no more calls expected: got LookupGroup(%v)", name)
	}
	if funcCall.LookupGroup == nil {
		t.T.Fatalf("unexpected call: got LookupGroup(%v), expected %#v", name, funcCall)
	}
	return funcCall.LookupGroup.ReturnGroup, funcCall.LookupGroup.ReturnError
}

func (t TestHost) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	logger := log.MustLoggerIndented(ctx)
	logger.Debug("TestHost.Lstat", "name", name)
	funcCall := t.getFnCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: got Lstat(%v)", name)
	}
	if funcCall.Lstat == nil {
		t.T.Fatalf("unexpected call: got Lstat(%v), expected %#v", name, funcCall)
	}
	return funcCall.Lstat.ReturnFileInfo, funcCall.Lstat.ReturnError
}

func (t TestHost) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.MustLoggerIndented(ctx)
	logger.Debug("TestHost.ReadFile", "name", name)
	funcCall := t.getFnCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: got ReadFile(%v)", name)
	}
	if funcCall.ReadFile == nil {
		t.T.Fatalf("unexpected call: got ReadFile(%v), expected %#v", name, funcCall)
	}
	return funcCall.ReadFile.ReturnBytes, funcCall.ReadFile.ReturnError
}

func (t TestHost) Remove(ctx context.Context, name string) error {
	logger := log.MustLoggerIndented(ctx)
	logger.Debug("TestHost.Remove", "name", name)
	funcCall := t.getFnCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: got Remove(%v)", name)
	}
	if funcCall.Remove == nil {
		t.T.Fatalf("unexpected call: got Remove(%v), expected %#v", name, funcCall)
	}
	return funcCall.Remove.ReturnError
}

func (t TestHost) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, string, string, error) {
	logger := log.MustLoggerIndented(ctx)
	logger.Debug("TestHost.Run", "cmd", cmd)
	funcCall := t.getFnCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: got Run(%v)", cmd)
	}
	if funcCall.Run == nil {
		t.T.Fatalf("unexpected call: got Run(%v), expected %#v", cmd, funcCall)
	}
	return funcCall.Run.ReturnWaitStatus, funcCall.Run.ReturnStdout, funcCall.Run.ReturnStderr, funcCall.Run.ReturnError
}

func (t TestHost) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	logger := log.MustLoggerIndented(ctx)
	logger.Debug("TestHost.WriteFile", "name", name, "data", string(data), "perm", perm)
	funcCall := t.getFnCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: got WriteFile(%v, %v, %v)", name, data, perm)
	}
	if funcCall.WriteFile == nil {
		t.T.Fatalf("unexpected call: got WriteFile(%v, %v, %v), expected %#v", name, data, perm, funcCall)
	}
	return funcCall.WriteFile.ReturnError
}

func (t TestHost) String() string {
	return "test"
}
