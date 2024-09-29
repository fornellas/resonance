package host

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"gopkg.in/yaml.v3"

	"github.com/fornellas/resonance/host"
	"github.com/fornellas/resonance/internal/host/agent_server_http/api"

	"github.com/fornellas/resonance/log"
)

var AgentHttpBinGz = map[string][]byte{}

// AgentHttpClient interacts with a given Host using an agent that's copied and ran at the
// host.
type AgentHttpClient struct {
	Host   host.Host
	path   string
	Client *http.Client
	waitCn chan struct{}
}

func NewHttpAgent(ctx context.Context, hst host.Host) (*AgentHttpClient, error) {
	ctx, _ = log.MustContextLoggerSection(ctx, "🐈 Agent")

	agentPath, err := getTmpFile(ctx, hst, "resonance_agent")
	if err != nil {
		return nil, err
	}

	if err := hst.Chmod(ctx, agentPath, 0755); err != nil {
		return nil, err
	}

	agentBinGz, err := getAgentBinGz(ctx, hst)
	if err != nil {
		return nil, err
	}

	agentReader, err := gzip.NewReader(bytes.NewReader(agentBinGz))
	if err != nil {
		return nil, err
	}

	if err := copyReader(ctx, hst, agentReader, agentPath); err != nil {
		return nil, err
	}

	agent := AgentHttpClient{
		Host:   hst,
		path:   agentPath,
		waitCn: make(chan struct{}),
	}

	if err := agent.spawn(ctx); err != nil {
		return nil, err
	}

	return &agent, nil
}

func (a AgentHttpClient) checkResponseStatus(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	} else if resp.StatusCode == http.StatusInternalServerError {
		decoder := yaml.NewDecoder(resp.Body)
		decoder.KnownFields(true)
		var apiErr api.Error
		if err := decoder.Decode(&apiErr); err != nil {
			return err
		}
		return apiErr.Error()
	} else {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unexpected status code %d: failed to read body: %s", resp.StatusCode, err)
		}
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

func (a AgentHttpClient) unmarshalResponse(resp *http.Response, bodyInterface interface{}) error {
	decoder := yaml.NewDecoder(resp.Body)
	decoder.KnownFields(true)
	if err := decoder.Decode(bodyInterface); err != nil {
		return err
	}
	return nil
}

func (a AgentHttpClient) get(path string) (*http.Response, error) {
	resp, err := a.Client.Get(fmt.Sprintf("http://agent%s", path))
	if err != nil {
		return nil, err
	}

	if err := a.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a AgentHttpClient) post(path string, bodyInterface interface{}) (*http.Response, error) {
	url := fmt.Sprintf("http://agent%s", path)

	contentType := "application/yaml"

	bodyData, err := yaml.Marshal(bodyInterface)
	if err != nil {
		return nil, err
	}
	body := bytes.NewBuffer(bodyData)

	resp, err := a.Client.Post(url, contentType, body)
	if err != nil {
		return nil, err
	}

	return resp, a.checkResponseStatus(resp)
}

func (a AgentHttpClient) delete(path string) (*http.Response, error) {
	url := fmt.Sprintf("http://agent%s", path)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if err := a.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a AgentHttpClient) put(path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("http://agent%s", path)
	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if err := a.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a AgentHttpClient) Chmod(ctx context.Context, name string, mode uint32) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Chmod", "name", name, "mode", mode)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	_, err := a.post(fmt.Sprintf("/file%s", name), api.File{
		Action: api.Chmod,
		Mode:   mode,
	})

	return err
}

func (a AgentHttpClient) Chown(ctx context.Context, name string, uid, gid int) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Chown", "name", name, "uid", uid, "gid", gid)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	_, err := a.post(fmt.Sprintf("/file%s", name), api.File{
		Action: api.Chown,
		Uid:    uid,
		Gid:    gid,
	})

	return err
}

func (a AgentHttpClient) Lookup(ctx context.Context, username string) (*user.User, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lookup", "username", username)

	resp, err := a.get(fmt.Sprintf("/user/%s", username))
	if err != nil {
		return nil, err
	}

	var u user.User
	if err := a.unmarshalResponse(resp, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (a AgentHttpClient) LookupGroup(ctx context.Context, name string) (*user.Group, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("LookupGroup", "name", name)

	resp, err := a.get(fmt.Sprintf("/group/%s", name))
	if err != nil {
		return nil, err
	}

	var g user.Group
	if err := a.unmarshalResponse(resp, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

func (a AgentHttpClient) Lstat(ctx context.Context, name string) (*syscall.Stat_t, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Lstat", "name", name)
	panic("TODO AgentHttpClient.Lstat")

	// if !filepath.IsAbs(name) {
	// 	return host.HostFileInfo{}, fmt.Errorf("path must be absolute: %s", name)
	// }

	// resp, err := a.get(fmt.Sprintf("/file%s?lstat=true", name))
	// if err != nil {
	// 	return host.HostFileInfo{}, err
	// }

	// var hfi host.HostFileInfo
	// if err := a.unmarshalResponse(resp, &hfi); err != nil {
	// 	return host.HostFileInfo{}, err
	// }
	// hfi.ModTime = hfi.ModTime.Local()
	// return hfi, nil
}

func (a AgentHttpClient) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Mkdir", "name", name)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	_, err := a.post(fmt.Sprintf("/file%s", name), api.File{
		Action: api.Mkdir,
		Mode:   uint32(perm),
	})

	return err
}

func (a AgentHttpClient) ReadFile(ctx context.Context, name string) ([]byte, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("ReadFile", "name", name)

	if !filepath.IsAbs(name) {
		return nil, fmt.Errorf("path must be absolute: %s", name)
	}

	resp, err := a.get(fmt.Sprintf("/file%s", name))
	if err != nil {
		return nil, err
	}

	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return contents, nil
}

func (a AgentHttpClient) Remove(ctx context.Context, name string) error {
	logger := log.MustLogger(ctx)

	logger.Debug("Remove", "name", name)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	_, err := a.delete(fmt.Sprintf("/file%s", name))
	if err != nil {
		return err
	}

	return nil
}

func (a AgentHttpClient) Run(ctx context.Context, cmd host.Cmd) (host.WaitStatus, error) {
	logger := log.MustLogger(ctx)

	logger.Debug("Run", "cmd", cmd)

	var stdin []byte
	if cmd.Stdin != nil {
		var err error
		stdin, err = io.ReadAll(cmd.Stdin)
		if err != nil {
			return host.WaitStatus{}, err
		}
	}

	var stdout bool
	if cmd.Stdout != nil {
		stdout = true
	}

	var stderr bool
	if cmd.Stderr != nil {
		stderr = true
	}

	resp, err := a.post("/run", api.Cmd{
		Path:   cmd.Path,
		Args:   cmd.Args,
		Env:    cmd.Env,
		Dir:    cmd.Dir,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return host.WaitStatus{}, err
	}

	var cs api.CmdResponse
	if err := a.unmarshalResponse(resp, &cs); err != nil {
		return host.WaitStatus{}, err
	}

	if cmd.Stdout != nil {
		_, err := io.Copy(cmd.Stdout, bytes.NewReader(cs.Stdout))
		if err != nil {
			return host.WaitStatus{}, err
		}
	}

	if cmd.Stderr != nil {
		_, err := io.Copy(cmd.Stderr, bytes.NewReader(cs.Stderr))
		if err != nil {
			return host.WaitStatus{}, err
		}
	}

	return cs.WaitStatus, nil
}

func (a AgentHttpClient) WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	logger := log.MustLogger(ctx)

	logger.Debug("WriteFile", "name", name, "data", data, "perm", perm)

	if !filepath.IsAbs(name) {
		return fmt.Errorf("path must be absolute: %s", name)
	}

	_, err := a.put(fmt.Sprintf("/file%s?perm=%d", name, perm), bytes.NewReader(data))
	if err != nil {
		return err
	}

	return nil
}

func (a AgentHttpClient) String() string {
	return a.Host.String()
}

func (a AgentHttpClient) Type() string {
	return a.Host.Type()
}

func (a *AgentHttpClient) Close() error {
	a.post("/shutdown", nil)
	a.Client.CloseIdleConnections()
	<-a.waitCn
	return a.Host.Close()
}
