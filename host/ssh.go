package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/alessio/shellescape"

	"github.com/fornellas/resonance/log"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Ssh interacts with a remote machine connecting to it via SSH protocol.
type Ssh struct {
	baseRun
	Hostname string
	client   *ssh.Client
}

func (s Ssh) Run(ctx context.Context, cmd Cmd) (WaitStatus, string, string, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Run %s", cmd)

	session, err := s.client.NewSession()
	if err != nil {
		return WaitStatus{}, "", "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	session.Stdin = cmd.Stdin
	var stdoutBuffer bytes.Buffer
	session.Stdout = &stdoutBuffer
	var stderrBuffer bytes.Buffer
	session.Stderr = &stderrBuffer

	var cmdStrBdr strings.Builder
	fmt.Fprintf(&cmdStrBdr, "%s", shellescape.Quote(cmd.Path))
	for _, arg := range cmd.Args {
		fmt.Fprintf(&cmdStrBdr, " %s", shellescape.Quote(arg))
	}

	var exitCode int
	var exited bool
	var signal string
	// TODO cmd.Env
	// TODO cmd.Dir
	if err := session.Run(cmdStrBdr.String()); err == nil {
		exitCode = 0
		exited = true
	} else {
		if errors.Is(err, &ssh.ExitError{}) {
			exitError := err.(*ssh.ExitError)
			exitCode = exitError.ExitStatus()
			exited = exitError.Signal() == ""
			signal = exitError.Signal()
		} else {
			return WaitStatus{}, "", "", fmt.Errorf(
				"failed to run %v: %w\nstdout:\n%s\nstderr:\n%s",
				cmd, err, stdoutBuffer.String(), stderrBuffer.String(),
			)
		}
	}

	return WaitStatus{
		ExitCode: exitCode,
		Exited:   exited,
		Signal:   signal,
	}, stdoutBuffer.String(), stderrBuffer.String(), nil
}

func (s Ssh) String() string {
	return s.Hostname
}

func (s Ssh) Close() error {
	return s.client.Close()
}

func sshKeyboardInteractiveChallenge(
	name, instruction string,
	questions []string,
	echos []bool,
) ([]string, error) {
	answers := make([]string, len(questions))
	var err error

	fmt.Printf("\nName: %s\nInstruction: %s\n", name, instruction)

	for i, question := range questions {
		if echos[i] {
			fmt.Printf("%s: ", question)
			_, _ = fmt.Scan(&answers[i])
		} else {
			var answerBytes []byte
			fmt.Printf("%s (no echo): ", question)
			answerBytes, err = (term.ReadPassword(int(os.Stdin.Fd())))
			if err != nil {
				return nil, err
			}
			fmt.Printf("\n\r")
			answers[i] = string(answerBytes)
		}
	}

	return answers, err
}

func sshGetPasswordCallbackPromptFn(user, host string) func() (secret string, err error) {
	return func() (secret string, err error) {
		state, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		defer term.Restore(int(os.Stdin.Fd()), state)

		fmt.Printf("Password for %s@%s: ", user, host)
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		fmt.Print("\n\r")
		return string(password), nil
	}
}

func sshGetSigners() ([]ssh.Signer, error) {
	signers := []ssh.Signer{}
	home, err := os.UserHomeDir()
	if err != nil {
		return signers, err
	}
	privateKeyPath := filepath.Join(home, ".ssh/id_rsa")
	privateKey, err := os.ReadFile(privateKeyPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return signers, fmt.Errorf("unable to read %s: %w", privateKeyPath, err)
		}
	} else {
		signer, err := ssh.ParsePrivateKey(privateKey)
		if err != nil {
			return signers, fmt.Errorf("unable to parse %s: %v", privateKeyPath, err)
		}
		signers = append(signers, signer)
	}
	return signers, nil
}

func sshGetHostKeyCallback(fingerprint string) (ssh.HostKeyCallback, error) {
	var fingerprintHostKeyCallback ssh.HostKeyCallback
	if fingerprint != "" {
		publicKey, err := ssh.ParsePublicKey([]byte(fingerprint))
		if err != nil {
			return nil, fmt.Errorf("fail to parse fingerprint: %w", err)
		}
		fingerprintHostKeyCallback = ssh.FixedHostKey(publicKey)
	}

	files := []string{}
	systemKnownHosts := "/etc/ssh/ssh_known_hosts"
	if _, err := os.Stat(systemKnownHosts); err == nil {
		files = append(files, systemKnownHosts)
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	userKnownHosts := filepath.Join(home, ".ssh/known_hosts")
	if _, err := os.Stat(userKnownHosts); err == nil {
		files = append(files, userKnownHosts)
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	knownHostsHostKeyCallback, err := knownhosts.New(files...)
	if err != nil {
		return nil, err
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if fingerprintHostKeyCallback != nil {
			if err := fingerprintHostKeyCallback(hostname, remote, key); err == nil {
				return nil
			}
		}
		return knownHostsHostKeyCallback(hostname, remote, key)
	}, nil
}

func NewSsh(
	ctx context.Context,
	user,
	fingerprint,
	host string,
	port int,
	timeout time.Duration,
) (Ssh, error) {
	signers, err := sshGetSigners()
	if err != nil {
		return Ssh{}, err
	}
	hostKeyCallback, err := sshGetHostKeyCallback(fingerprint)
	if err != nil {
		return Ssh{}, err
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			// ssh.GSSAPIWithMICAuthMethod(gssAPIClient GSSAPIClient, target string)
			ssh.KeyboardInteractive(sshKeyboardInteractiveChallenge),
			ssh.PasswordCallback(sshGetPasswordCallbackPromptFn(user, host)),
			ssh.PublicKeys(signers...),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	})
	if err != nil {
		return Ssh{}, fmt.Errorf("failed to connect: %w", err)
	}

	sshHost := Ssh{
		Hostname: host,
		client:   client,
	}
	sshHost.baseRun.Host = sshHost

	return sshHost, nil
}

var authorityRegexp = regexp.MustCompile(`^(?:(?P<user>[^;@]+)(?:;fingerprint=(?P<fingerprint>[^@]+))?)?@(?P<host>[^:]+)(?::(?P<port>\d+))?$`)

func parseAuthority(authority string) (string, string, string, int, error) {
	matches := authorityRegexp.FindStringSubmatch(authority)
	if matches == nil {
		return "", "", "", 0, errors.New(
			"invalid URI format, it must match [<user>[;fingerprint=<host-key fingerprint>]@]<host>[:<port>]",
		)
	}
	usr := matches[authorityRegexp.SubexpIndex("user")]
	fingerprint := matches[authorityRegexp.SubexpIndex("fingerprint")]
	host := matches[authorityRegexp.SubexpIndex("host")]
	port := 22
	portStr := matches[authorityRegexp.SubexpIndex("port")]
	if portStr != "" {
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return "", "", "", 0, fmt.Errorf("invalid port number: %w", err)
		}
	}

	return usr, fingerprint, host, port, nil
}

var DefaultSshTCPConnectTimeout = time.Second * 30

// NewSshAuthority creates a new Ssh from given authority in the format
// [<user>[;fingerprint=<host-key fingerprint>]@]<host>[:<port>]
// based on https://www.iana.org/assignments/uri-schemes/prov/ssh
func NewSshAuthority(ctx context.Context, authority string) (Ssh, error) {
	user, fingerprint, host, port, err := parseAuthority(authority)
	if err != nil {
		return Ssh{}, err
	}
	return NewSsh(ctx, user, fingerprint, host, port, DefaultSshTCPConnectTimeout)
}
