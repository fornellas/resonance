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

type IndividualFuncChmod struct {
	Name        string
	FileModeode os.FileMode
	ReturnError error
}

type IndividualFuncChown struct {
	Name        string
	Uid         int
	Gid         int
	ReturnError error
}

type IndividualFuncLookup struct {
	Username    string
	ReturnUser  *user.User
	ReturnError error
}

type IndividualFuncLookupGroup struct {
	Name        string
	ReturnGroup *user.Group
	ReturnError error
}

type IndividualFuncLstat struct {
	Name           string
	ReturnFileInfo os.FileInfo
	ReturnError    error
}

type IndividualFuncReadFile struct {
	Name        string
	ReturnBytes []byte
	ReturnError error
}

type IndividualFuncRemove struct {
	Name        string
	ReturnError error
}

type IndividualFuncRun struct {
	Cmd              host.Cmd
	ReturnWaitStatus host.WaitStatus
	ReturnStdout     string
	ReturnStderr     string
	ReturnError      error
}

type IndividualFuncWriteFile struct {
	Name        string
	Data        []byte
	Perm        os.FileMode
	ReturnError error
}

type IndividualFuncCall struct {
	Chmod       *IndividualFuncChmod
	Chown       *IndividualFuncChown
	Lookup      *IndividualFuncLookup
	LookupGroup *IndividualFuncLookupGroup
	Lstat       *IndividualFuncLstat
	ReadFile    *IndividualFuncReadFile
	Remove      *IndividualFuncRemove
	Run         *IndividualFuncRun
	WriteFile   *IndividualFuncWriteFile
}

// Test aids testing by enabling mocking of host functions.
type Test struct {
	T                           *testing.T
	ExpectedIndividualFuncCalls []IndividualFuncCall
}

func (t *Test) getFuncCall() *IndividualFuncCall {
	if len(t.ExpectedIndividualFuncCalls) == 0 {
		t.T.Fail() // FIXME use T.Fatalf()
		return nil
	}
	testFuncCall, expectedIndividualFuncCalls := t.ExpectedIndividualFuncCalls[0], t.ExpectedIndividualFuncCalls[1:]
	t.ExpectedIndividualFuncCalls = expectedIndividualFuncCalls
	return &testFuncCall
}

func (t Test) Chmod(ctx context.Context, name string, mode os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Chmod %v %s", mode, name)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		return fmt.Errorf("no more calls expected: Chmod(%v, %v)", name, mode)
	}
	if funcCall.Chmod == nil {
		t.T.Fail() // FIXME use T.Fatalf()
		return fmt.Errorf("unexpected call: got Chmod(%v, %v), expected %#v", name, mode, funcCall)
	}
	return funcCall.Chmod.ReturnError
}

func (t Test) Chown(ctx context.Context, name string, uid, gid int) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Chown %v %v %s", uid, gid, name)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		return fmt.Errorf("no more calls expected: Chown(%v, %v, %v)", name, uid, gid)
	}
	if funcCall.Chown == nil {
		t.T.Fail() // FIXME use T.Fatalf()
		return fmt.Errorf("unexpected call: got Chown(%v, %v, %v), expected %#v", name, uid, gid, funcCall)
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
		t.T.Fail() // FIXME use T.Fatalf()
		return nil, fmt.Errorf("unexpected call: got Lookup(%v), expected %#v", username, funcCall)
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
		t.T.Fail() // FIXME use T.Fatalf()
		return nil, fmt.Errorf("unexpected call: got LookupGroup(%v), expected %#v", name, funcCall)
	}
	return funcCall.LookupGroup.ReturnGroup, funcCall.LookupGroup.ReturnError
}

func (t Test) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Lstat %s", name)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		return nil, fmt.Errorf("no more calls expected: got Lstat(%v)", name)
	}
	if funcCall.Lstat == nil {
		t.T.Fail() // FIXME use T.Fatalf()
		return nil, fmt.Errorf("unexpected call: got Lstat(%v), expected %#v", name, funcCall)
	}
	return funcCall.Lstat.ReturnFileInfo, funcCall.Lstat.ReturnError
}

func (t Test) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("ReadFile %s", name)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		return nil, fmt.Errorf("no more calls expected: got ReadFile(%v)", name)
	}
	if funcCall.ReadFile == nil {
		t.T.Fail() // FIXME use T.Fatalf()
		return nil, fmt.Errorf("unexpected call: got ReadFile(%v), expected %#v", name, funcCall)
	}
	return funcCall.ReadFile.ReturnBytes, funcCall.ReadFile.ReturnError
}

func (t Test) Remove(ctx context.Context, name string) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("Remove %s", name)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		return fmt.Errorf("no more calls expected: got Remove(%v)", name)
	}
	if funcCall.Remove == nil {
		t.T.Fail() // FIXME use T.Fatalf()
		return fmt.Errorf("unexpected call: got Remove(%v), expected %#v", name, funcCall)
	}
	return funcCall.Remove.ReturnError
}

func (t Test) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, string, string, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Run %s", cmd)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		return host.WaitStatus{}, "", "", fmt.Errorf("no more calls expected: got Run(%v)", cmd)
	}
	if funcCall.Run == nil {
		t.T.Fail() // FIXME use T.Fatalf()
		return host.WaitStatus{}, "", "", fmt.Errorf("unexpected call: got Run(%v), expected %#v", cmd, funcCall)
	}
	return funcCall.Run.ReturnWaitStatus, funcCall.Run.ReturnStdout, funcCall.Run.ReturnStderr, funcCall.Run.ReturnError
}

func (t Test) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	logger := log.GetLogger(ctx)
	logger.Debugf("WriteFile %s %v", name, perm)
	funcCall := t.getFuncCall()
	if funcCall == nil {
		return fmt.Errorf("no more calls expected: got WriteFile(%v, %v, %v)", name, data, perm)
	}
	if funcCall.WriteFile == nil {
		t.T.Fail() // FIXME use T.Fatalf()
		return fmt.Errorf("unexpected call: got WriteFile(%v, %v, %v), expected %#v", name, data, perm, funcCall)
	}
	return funcCall.WriteFile.ReturnError
}

func (t Test) String() string {
	return "test"
}
