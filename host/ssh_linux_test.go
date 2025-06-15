package host

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/gliderlabs/ssh"
	"github.com/stretchr/testify/require"
	goSsh "golang.org/x/crypto/ssh"

	"github.com/fornellas/slogxt/log"
)

func getUsername() string {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])
}

func getSshHandler(t *testing.T, username string) func(session ssh.Session) {
	return func(session ssh.Session) {
		if session.User() != username {
			t.Fatalf("bad username %s", username)
		}
		if len(session.Subsystem()) > 0 {
			t.Fatalf("unexpected Subsystem %#v", session.Subsystem())
		}
		if len(session.Command()) == 0 {
			t.Fatalf("shell not supported")
		}

		cmd := exec.Command(session.Command()[0], session.Command()[1:]...)
		cmd.Env = append(os.Environ(), session.Environ()...)
		cmd.Stdin = session
		cmd.Stdout = session
		cmd.Stderr = session.Stderr()
		err := cmd.Run()
		if err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				fmt.Fprintf(session.Stderr(), "%s", err)
				session.Close()
				return
			}
		}
		if cmd.ProcessState.Exited() {
			session.Exit(cmd.ProcessState.ExitCode())
		} else {
			fmt.Fprintf(session.Stderr(), "%s", err)
			session.Close()
		}
	}
}

func TestSsh(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithTestLogger(ctx)

	listener, err := net.Listen("tcp4", "localhost:")
	require.NoError(t, err)
	addrChunks := strings.Split(listener.Addr().String(), ":")
	require.Len(t, addrChunks, 2)
	port, err := strconv.ParseInt(addrChunks[1], 10, 32)
	require.NoError(t, err)

	serverPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	serverSigner, err := goSsh.NewSignerFromKey(serverPrivateKey)
	require.NoError(t, err)
	serverFingerprint := goSsh.FingerprintSHA256(serverSigner.PublicKey())

	username := getUsername()

	server := &ssh.Server{
		Handler:     getSshHandler(t, username),
		HostSigners: []ssh.Signer{serverSigner},
	}
	go server.Serve(listener)
	defer func() { server.Close() }()

	authority := fmt.Sprintf("%s;fingerprint=%s@localhost:%d", username, serverFingerprint, port)
	baseHost, err := NewSshAuthority(ctx, authority, SshClientConfig{})
	require.NoError(t, err)
	defer func() { require.NoError(t, baseHost.Close(ctx)) }()

	tempDirPrefix := t.TempDir()
	testBaseHost(t, ctx, tempDirPrefix, baseHost, "localhost", "ssh")
}
