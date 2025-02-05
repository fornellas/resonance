package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"al.essio.dev/pkg/shellescape"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/log"
)

// Ssh interacts with a remote machine connecting to it via SSH protocol.
type Ssh struct {
	Hostname string
	client   *ssh.Client
	envPath  string
}

func sshGetSigners(ctx context.Context) ([]ssh.Signer, error) {
	logger := log.MustLogger(ctx)

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
			logger.Debug("No private key found", "path", privateKeyPath)
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
			logger.Debug("Using private key", "path", privateKeyPath)
			signers = append(signers, signer)
		}
	}
	return signers, nil
}

func sshGetHostKeyCallback(ctx context.Context, fingerprint string) (ssh.HostKeyCallback, error) {
	logger := log.MustLogger(ctx)

	var fingerprintHostKeyCallback ssh.HostKeyCallback
	if fingerprint != "" {
		if !strings.HasPrefix(fingerprint, "SHA256:") {
			return nil, fmt.Errorf(
				"fingerprint must be an unpadded base64 encoded sha256 hash as introduced by https://www.openssh.com/txt/release-6.8, eg: %s",
				"SHA256:uwhOoCVTS7b3wlX1popZs5k609OaD1vQurHU34cCWPk",
			)
		}
		logger.Debug("Using fingerprint")
		fingerprintHostKeyCallback = func(hostname string, remote net.Addr, hostPublicKey ssh.PublicKey) error {
			hostFingerprint := ssh.FingerprintSHA256(hostPublicKey)
			if fingerprint != hostFingerprint {
				return fmt.Errorf("expected host fingerprint %s, got %s", fingerprint, hostFingerprint)
			}
			return nil
		}
	}

	files := []string{}
	systemKnownHosts := "/etc/ssh/ssh_known_hosts"
	if _, err := os.Stat(systemKnownHosts); err == nil {
		logger.Debug("Using known hosts", "path", systemKnownHosts)
		files = append(files, systemKnownHosts)
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		logger.Debug("Known hosts not found", "path", systemKnownHosts)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	userKnownHosts := filepath.Join(home, ".ssh/known_hosts")
	if _, err := os.Stat(userKnownHosts); err == nil {
		logger.Debug("Using knwon hosts", "path", userKnownHosts)
		files = append(files, userKnownHosts)
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		logger.Debug("Known hosts not found", "path", userKnownHosts)
	}
	knownHostsHostKeyCallback, err := knownhosts.New(files...)
	if err != nil {
		return nil, err
	}

	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if fingerprintHostKeyCallback != nil {
			if err := fingerprintHostKeyCallback(hostname, remote, key); err == nil {
				logger.Debug("Server key verified by fingerprint")
				return nil
			}
		}
		err := knownHostsHostKeyCallback(hostname, remote, key)
		if err == nil {
			logger.Debug("Server key verified by known_hosts")
		}
		return err
	}

	return hostKeyCallback, nil
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

func NewSsh(
	ctx context.Context,
	user,
	fingerprint,
	host string,
	port int,
	timeout time.Duration,
) (Ssh, error) {
	ctx, _ = log.MustContextLoggerSection(
		ctx,
		"🖧 SSH",
		"user", user,
		"fingerprint", fingerprint,
		"host", host,
		"port", port,
		"timeout", timeout,
	)

	signers, err := sshGetSigners(ctx)
	if err != nil {
		return Ssh{}, err
	}
	hostKeyCallback, err := sshGetHostKeyCallback(ctx, fingerprint)
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

	if err := sshHost.setEnvPath(ctx); err != nil {
		return Ssh{}, err
	}

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

func (h Ssh) runEnv(ctx context.Context, cmd host.Cmd, ignoreCmdEnv bool) (host.WaitStatus, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Run", "cmd", cmd)

	session, err := h.client.NewSession()
	if err != nil {
		return host.WaitStatus{}, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	session.Stdin = cmd.Stdin
	session.Stdout = cmd.Stdout
	session.Stderr = cmd.Stderr

	shellCmdArgs := []string{shellescape.Quote(cmd.Path)}
	for _, arg := range cmd.Args {
		shellCmdArgs = append(shellCmdArgs, shellescape.Quote(arg))
	}
	shellCmdStr := strings.Join(shellCmdArgs, " ")

	if cmd.Dir == "" {
		cmd.Dir = "/tmp"
	}
	if !filepath.IsAbs(cmd.Dir) {
		return host.WaitStatus{}, &fs.PathError{
			Op:   "Run",
			Path: cmd.Dir,
			Err:  errors.New("path must be absolute"),
		}
	}

	var args []string
	if !ignoreCmdEnv {
		if len(cmd.Env) == 0 {
			cmd.Env = []string{"LANG=en_US.UTF-8"}
			if h.envPath != "" {
				cmd.Env = append(cmd.Env, h.envPath)
			}
		}
		envStrs := []string{}
		for _, nameValue := range cmd.Env {
			envStrs = append(envStrs, shellescape.Quote(nameValue))
		}
		envStr := strings.Join(envStrs, " ")
		args = []string{"sh", "-c", fmt.Sprintf(
			"cd %s && exec env --ignore-environment %s %s", shellescape.Quote(cmd.Dir), envStr, shellCmdStr,
		)}
	} else {
		args = []string{"sh", "-c", fmt.Sprintf(
			"cd %s && exec env %s", shellescape.Quote(cmd.Dir), shellCmdStr,
		)}
	}

	var cmdStrBdr strings.Builder
	fmt.Fprintf(&cmdStrBdr, "%s", shellescape.Quote(args[0]))
	for _, arg := range args[1:] {
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
			return host.WaitStatus{}, fmt.Errorf("failed to run %v: %w", cmd, err)
		}
	}

	return host.WaitStatus{
		ExitCode: exitCode,
		Exited:   exited,
		Signal:   signal,
	}, nil
}

func (h Ssh) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	return h.runEnv(ctx, cmd, false)
}

func (h *Ssh) setEnvPath(ctx context.Context) error {
	stdoutBuffer := bytes.Buffer{}
	stderrBuffer := bytes.Buffer{}
	cmd := host.Cmd{
		Path:   "env",
		Stdout: &stdoutBuffer,
		Stderr: &stderrBuffer,
	}
	waitStatus, err := h.runEnv(ctx, cmd, true)
	if err != nil {
		return err
	}
	if !waitStatus.Success() {
		return fmt.Errorf(
			"failed to run %s: %s\nstdout:\n%s\nstderr:\n%s",
			cmd, waitStatus.String(), stdoutBuffer.String(), stderrBuffer.String(),
		)
	}
	for _, value := range strings.Split(stdoutBuffer.String(), "\n") {
		if strings.HasPrefix(value, "PATH=") {
			h.envPath = value
			break
		}
	}
	return nil
}

func (h Ssh) String() string {
	return h.Hostname
}

func (h Ssh) Type() string {
	return "ssh"
}

func (h Ssh) Close(ctx context.Context) error {
	return h.client.Close()
}
