package host

import (
	"bytes"
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alessio/shellescape"

	"github.com/fornellas/resonance/log"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
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

	if err := session.RequestPty("xterm", 40, 80, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		return WaitStatus{}, "", "", fmt.Errorf("failed to request pty: %w", err)
	}

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

	if name != "" {
		fmt.Printf("Name: %s\n", name)
	}
	if instruction != "" {
		fmt.Printf("Instruction: %s\n", instruction)
	}

	for i, question := range questions {
		if echos[i] {
			fmt.Printf("%s: ", question)
			_, _ = fmt.Scan(&answers[i])
		} else {
			var answerBytes []byte
			fmt.Printf("%s", question)
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

func sshGetPasswordCallbackPromptFn() func() (secret string, err error) {
	return func() (secret string, err error) {
		state, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		defer term.Restore(int(os.Stdin.Fd()), state)

		fmt.Printf("Password: ")
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		fmt.Print("\n\r")
		return string(password), nil
	}
}

func sshGetSigners(ctx context.Context) ([]ssh.Signer, error) {
	logger := log.GetLogger(ctx)

	signers := []ssh.Signer{}
	home, err := os.UserHomeDir()
	if err != nil {
		return signers, err
	}

	for _, privateKeySuffix := range []string{
		".ssh/id_rsa",
		".ssh/id_ecdsa",
		".ssh/id_ecdsa_sk",
		".ssh/id_ed25519",
		".ssh/id_ed25519_sk",
		".ssh/id_dsa",
	} {
		privateKeyPath := filepath.Join(home, privateKeySuffix)
		privateKeyBytes, err := os.ReadFile(privateKeyPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return signers, fmt.Errorf("unable to read %s: %w", privateKeyPath, err)
			}
			logger.Debugf("No private key found at %s", privateKeyPath)
		} else {
			var signer ssh.Signer
			var err error
			signer, err = ssh.ParsePrivateKey(privateKeyBytes)
			if err != nil {
				if errors.Is(err, &ssh.PassphraseMissingError{}) {
					state, err := term.MakeRaw(int(os.Stdin.Fd()))
					if err != nil {
						return nil, err
					}
					defer term.Restore(int(os.Stdin.Fd()), state)

					fmt.Printf("Password for %s: ", privateKeyPath)
					passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
					if err != nil {
						return nil, err
					}
					fmt.Print("\n\r")

					signer, err = ssh.ParsePrivateKeyWithPassphrase(privateKeyBytes, passphrase)
					if err != nil {
						return signers, fmt.Errorf("unable to parse %s: %v", privateKeyPath, err)
					}
				} else {
					return signers, fmt.Errorf("unable to parse %s: %v", privateKeyPath, err)
				}
			}
			logger.Debugf("Using private key %s", privateKeyPath)
			signers = append(signers, signer)
		}
	}
	return signers, nil
}

func getHostKeyAlgorithms(
	host string, port int, knownHostsHostKeyCallback ssh.HostKeyCallback,
) []string {
	// https://github.com/golang/go/issues/29286#issuecomment-1160958614
	hostKeyAlgorithms := []string{}
	var keyErr *knownhosts.KeyError
	publicKey, err := ssh.NewPublicKey(&rsa.PublicKey{
		N: big.NewInt(0),
	})
	if err != nil {
		panic(err)
	}
	err = knownHostsHostKeyCallback(
		fmt.Sprintf("%s:%d", host, port),
		&net.TCPAddr{IP: []byte{0, 0, 0, 0}},
		publicKey,
	)
	if errors.As(err, &keyErr) {
		knownKeys := append([]knownhosts.KnownKey{}, keyErr.Want...)

		knownKeysLess := func(i, j int) bool {
			if knownKeys[i].Filename < knownKeys[j].Filename {
				return true
			}
			return (knownKeys[i].Filename == knownKeys[j].Filename && knownKeys[i].Line < knownKeys[j].Line)
		}
		sort.Slice(knownKeys, knownKeysLess)

		for _, knownKey := range keyErr.Want {
			hostKeyAlgorithms = append(hostKeyAlgorithms, knownKey.Key.Type())
		}
	}
	return hostKeyAlgorithms
}

func sshGetHostKeyCallback(
	ctx context.Context, host string, port int, fingerprint string,
) (ssh.HostKeyCallback, []string, error) {
	logger := log.GetLogger(ctx)
	var fingerprintHostKeyCallback ssh.HostKeyCallback
	if fingerprint != "" {
		publicKey, err := ssh.ParsePublicKey([]byte(fingerprint))
		if err != nil {
			return nil, nil, fmt.Errorf("fail to parse fingerprint: %w", err)
		}
		logger.Debug("Using fingerprint")
		fingerprintHostKeyCallback = ssh.FixedHostKey(publicKey)
	}

	files := []string{}
	systemKnownHosts := "/etc/ssh/ssh_known_hosts"
	if _, err := os.Stat(systemKnownHosts); err == nil {
		logger.Debugf("Using %s", systemKnownHosts)
		files = append(files, systemKnownHosts)
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, nil, err
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	userKnownHosts := filepath.Join(home, ".ssh/known_hosts")
	if _, err := os.Stat(userKnownHosts); err == nil {
		logger.Debugf("Using %s", userKnownHosts)
		files = append(files, userKnownHosts)
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, nil, err
		}
	}
	knownHostsHostKeyCallback, err := knownhosts.New(files...)
	if err != nil {
		return nil, nil, err
	}

	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if fingerprintHostKeyCallback != nil {
			if err := fingerprintHostKeyCallback(hostname, remote, key); err == nil {
				logger.Debugf("Server key verified by fingerprint")
				return nil
			}
		}
		err := knownHostsHostKeyCallback(hostname, remote, key)
		if err == nil {
			logger.Debugf("Server key verified by known_hosts")
		}
		return err
	}

	hostKeyAlgorithms := getHostKeyAlgorithms(host, port, knownHostsHostKeyCallback)

	return hostKeyCallback, hostKeyAlgorithms, nil
}

func NewSsh(
	ctx context.Context,
	user,
	fingerprint,
	host string,
	port int,
	timeout time.Duration,
) (Ssh, error) {
	logger := log.GetLogger(ctx)
	signers, err := sshGetSigners(ctx)
	if err != nil {
		return Ssh{}, err
	}
	hostKeyCallback, hostKeyAlgorithms, err := sshGetHostKeyCallback(ctx, host, port, fingerprint)
	if err != nil {
		return Ssh{}, err
	}
	logger.Debugf("Host key algorithms: %v", hostKeyAlgorithms)
	retries := 3
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signers...),
			ssh.RetryableAuthMethod(ssh.PasswordCallback(sshGetPasswordCallbackPromptFn()), retries),
			ssh.RetryableAuthMethod(ssh.KeyboardInteractive(sshKeyboardInteractiveChallenge), retries),
		},
		HostKeyCallback:   hostKeyCallback,
		HostKeyAlgorithms: hostKeyAlgorithms,
		Timeout:           timeout,
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
