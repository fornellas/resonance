package host

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"testing"

	"github.com/fornellas/resonance/host/types"
	"github.com/fornellas/resonance/log"
)

type FuncChmod struct {
	Name        string
	FileModeode os.FileMode
	ReturnError error
}

type FuncChown struct {
	Name        string
	Uid         int
	Gid         int
	ReturnError error
}

type FuncLookup struct {
	Username    string
	ReturnUser  *user.User
	ReturnError error
}

type FuncLookupGroup struct {
	Name        string
	ReturnGroup *user.Group
	ReturnError error
}

type FuncLstat struct {
	Name           string
	ReturnFileInfo os.FileInfo
	ReturnError    error
}

type FuncReadFile struct {
	Name        string
	ReturnBytes []byte
	ReturnError error
}

type FuncRemove struct {
	Name        string
	ReturnError error
}

type FuncRun struct {
	Cmd              types.Cmd
	ReturnWaitStatus types.WaitStatus
	ReturnStdout     string
	ReturnStderr     string
	ReturnError      error
}

type FuncWriteFile struct {
	Name        string
	Data        []byte
	Perm        os.FileMode
	ReturnError error
}

type FuncCall struct {
	Chmod       *FuncChmod
	Chown       *FuncChown
	Lookup      *FuncLookup
	LookupGroup *FuncLookupGroup
	Lstat       *FuncLstat
	ReadFile    *FuncReadFile
	Remove      *FuncRemove
	Run         *FuncRun
	WriteFile   *FuncWriteFile
}

// Test aids testing by enabling mocking of host functions.
type Test struct {
	T                 *testing.T
	ExpectedFuncCalls []FuncCall
}

func (t *Test) getFuncCall() *FuncCall {
	if len(t.ExpectedFuncCalls) == 0 {
		return nil
	}
	testFuncCall, expectedFuncCalls := t.ExpectedFuncCalls[0], t.ExpectedFuncCalls[1:]
	t.ExpectedFuncCalls = expectedFuncCalls
	return &testFuncCall
}

func (t Test) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Chmod %v %s", mode, name)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: Chmod(%v, %v)", name, mode)
	}
	if funcCall.Chmod == nil {
		t.T.Fatalf("unexpected call: got Chmod(%v, %v), expected %#v", name, mode, funcCall)
	}
	return funcCall.Chmod.ReturnError
}

func (t Test) Chown(ctx context.Context, name string, uid, gid int) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Chown %v %v %s", uid, gid, name)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: Chown(%v, %v, %v)", name, uid, gid)
	}
	if funcCall.Chown == nil {
		t.T.Fatalf("unexpected call: got Chown(%v, %v, %v), expected %#v", name, uid, gid, funcCall)
	}
	return funcCall.Chown.ReturnError
}

func (t Test) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Lookup %s", username)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		return nil, fmt.Errorf("no more calls expected: got Lookup(%v)", username)
	}
	if funcCall.Lookup == nil {
		t.T.Fatalf("unexpected call: got Lookup(%v), expected %#v", username, funcCall)
	}
	return funcCall.Lookup.ReturnUser, funcCall.Lookup.ReturnError
}

func (t Test) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("LookupGroup %s", name)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		return nil, fmt.Errorf("no more calls expected: got LookupGroup(%v)", name)
	}
	if funcCall.LookupGroup == nil {
		t.T.Fatalf("unexpected call: got LookupGroup(%v), expected %#v", name, funcCall)
	}
	return funcCall.LookupGroup.ReturnGroup, funcCall.LookupGroup.ReturnError
}

func (t Test) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Lstat %s", name)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: got Lstat(%v)", name)
	}
	if funcCall.Lstat == nil {
		t.T.Fatalf("unexpected call: got Lstat(%v), expected %#v", name, funcCall)
	}
	return funcCall.Lstat.ReturnFileInfo, funcCall.Lstat.ReturnError
}

func (t Test) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("ReadFile %s", name)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: got ReadFile(%v)", name)
	}
	if funcCall.ReadFile == nil {
		t.T.Fatalf("unexpected call: got ReadFile(%v), expected %#v", name, funcCall)
	}
	return funcCall.ReadFile.ReturnBytes, funcCall.ReadFile.ReturnError
}

func (t Test) Remove(ctx context.Context, name string) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Remove %s", name)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: got Remove(%v)", name)
	}
	if funcCall.Remove == nil {
		t.T.Fatalf("unexpected call: got Remove(%v), expected %#v", name, funcCall)
	}
	return funcCall.Remove.ReturnError
}

func (t Test) Run(ctx context.Context, cmd types.Cmd) (types.WaitStatus, string, string, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Run %s", cmd)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: got Run(%v)", cmd)
	}
	if funcCall.Run == nil {
		t.T.Fatalf("unexpected call: got Run(%v), expected %#v", cmd, funcCall)
	}
	return funcCall.Run.ReturnWaitStatus, funcCall.Run.ReturnStdout, funcCall.Run.ReturnStderr, funcCall.Run.ReturnError
}

func (t Test) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("WriteFile %s %v", name, perm)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		t.T.Fatalf("no more calls expected: got WriteFile(%v, %v, %v)", name, data, perm)
	}
	if funcCall.WriteFile == nil {
		t.T.Fatalf("unexpected call: got WriteFile(%v, %v, %v), expected %#v", name, data, perm, funcCall)
	}
	return funcCall.WriteFile.ReturnError
}

func (t Test) String() string {
	return "test"
}
