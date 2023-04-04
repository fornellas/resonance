package host

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
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

func (s Ssh) Run(ctx context.Context, cmd Cmd) (WaitStatus, error) {
	logger := log.GetLogger(ctx)
	logger.Debugf("Run %s", cmd)

	session, err := s.client.NewSession()
	if err != nil {
		return WaitStatus{}, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	session.Stdin = cmd.Stdin
	session.Stdout = cmd.Stdout
	session.Stderr = cmd.Stderr

	env := cmd.Env
	if len(env) == 0 {
		env = []string{"LANG=en_US.UTF-8"}
	}
	for _, nameValue := range env {
		equalsIdx := strings.Index(nameValue, "=")
		if equalsIdx == -1 {
			return WaitStatus{}, fmt.Errorf("invalid environment: %s", nameValue)
		}
		name := nameValue[:equalsIdx]
		value := nameValue[equalsIdx+1:]
		if err := session.Setenv(name, value); err != nil {
			return WaitStatus{}, err
		}
	}

	var cmdStrBdr strings.Builder
	if cmd.Dir == "" {
		cmd.Dir = "/tmp"
	}
	fmt.Fprintf(&cmdStrBdr, "cd %s && %s", shellescape.Quote(cmd.Dir), shellescape.Quote(cmd.Path))
	for _, arg := range cmd.Args {
		fmt.Fprintf(&cmdStrBdr, " %s", shellescape.Quote(arg))
	}

	var exitCode int
	var exited bool
	var signal string
	if err := session.Run(cmdStrBdr.String()); err == nil {
		exitCode = 0
		exited = true
	} else {
		var exitError *ssh.ExitError
		if errors.As(err, &exitError) {
			exitCode = exitError.ExitStatus()
			exited = exitError.Signal() == ""
			signal = exitError.Signal()
		} else {
			return WaitStatus{}, fmt.Errorf("failed to run %v: %w", cmd, err)
		}
	}

	return WaitStatus{
		ExitCode: exitCode,
		Exited:   exited,
		Signal:   signal,
	}, nil
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
			state, err := term.MakeRaw(int(os.Stdin.Fd()))
			if err != nil {
				return nil, err
			}
			defer term.Restore(int(os.Stdin.Fd()), state)

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

func sshGetHostKeyCallback(
	ctx context.Context, host string, port int, fingerprint string,
) (ssh.HostKeyCallback, error) {
	logger := log.GetLogger(ctx)
	var fingerprintHostKeyCallback ssh.HostKeyCallback
	if fingerprint != "" {
		publicKey, err := ssh.ParsePublicKey([]byte(fingerprint))
		if err != nil {
			return nil, fmt.Errorf("fail to parse fingerprint: %w", err)
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
			return nil, err
		}
		logger.Debugf("Not found %s", systemKnownHosts)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	userKnownHosts := filepath.Join(home, ".ssh/known_hosts")
	if _, err := os.Stat(userKnownHosts); err == nil {
		logger.Debugf("Using %s", userKnownHosts)
		files = append(files, userKnownHosts)
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		logger.Debugf("Not found %s", userKnownHosts)
	}
	knownHostsHostKeyCallback, err := knownhosts.New(files...)
	if err != nil {
		return nil, err
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

	return hostKeyCallback, nil
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
	logger.Infof("🖧 SSH %s@%s:%d", user, host, port)
	nestedCtx := log.IndentLogger(ctx)

	signers, err := sshGetSigners(nestedCtx)
	if err != nil {
		return Ssh{}, err
	}
	hostKeyCallback, err := sshGetHostKeyCallback(nestedCtx, host, port, fingerprint)
	if err != nil {
		return Ssh{}, err
	}

	retries := 3
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signers...),
			ssh.RetryableAuthMethod(ssh.PasswordCallback(sshGetPasswordCallbackPromptFn()), retries),
			ssh.RetryableAuthMethod(ssh.KeyboardInteractive(sshKeyboardInteractiveChallenge), retries),
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

var authorityRegexp = regexp.MustCompile(`(|((?P<user>[^;@]+)(|;fingerprint=(?P<fingerprint>[^@]+))@))(?P<host>[^:|@]+)(|:(?P<port>[0-9]+))$`)

func parseAuthority(authority string) (string, string, string, int, error) {
	matches := authorityRegexp.FindStringSubmatch(authority)
	if matches == nil {
		return "", "", "", 0, errors.New(
			"invalid URI format, it must match [<user>[;fingerprint=<host-key fingerprint>]@]<host>[:<port>]",
		)
	}
	usr := matches[authorityRegexp.SubexpIndex("user")]
	if usr == "" {
		currentUser, err := user.Current()
		if err != nil {
			return "", "", "", 0, err
		}
		usr = currentUser.Username
	}
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
