package bao

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

type ConfigBuilder interface {
	ToConfig(directory string) (string, error)
}

type ArgBuilder interface {
	AddArgs(directory string) ([]string, error)
}

const TLS_CERTS_NAME = "fullchain.pem"
const TLS_KEY_NAME = "leaf-key.pem"

type TLSConfig struct {
	Certificates []string `json:"certs"`
	Key          string   `json:"key"`
}

func (t *TLSConfig) Write(certPath string, keyPath string) error {
	certFile, err := os.OpenFile(certPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open certs to path (%v): %w", certPath, err)
	}
	defer certFile.Close()

	for index, cert := range t.Certificates {
		if _, err := io.WriteString(certFile, strings.TrimSpace(cert)+"\n"); err != nil {
			return fmt.Errorf("failed to write cert %d to path (%v): %w", index, certPath, err)
		}
	}

	keyFile, err := os.OpenFile(keyPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open keys to path (%v): %w", keyPath, err)
	}
	defer keyFile.Close()

	if _, err := io.WriteString(keyFile, strings.TrimSpace(t.Key)+"\n"); err != nil {
		return fmt.Errorf("failed to write key to path (%v): %w", keyPath, err)
	}

	return nil
}

func (t *TLSConfig) GetCA() (*x509.Certificate, string, error) {
	if len(t.Certificates) == 1 {
		return nil, "", nil
	}

	caPem := t.Certificates[len(t.Certificates)-1]
	caBlock, rest := pem.Decode([]byte(caPem))
	if len(rest) != 0 {
		return nil, "", fmt.Errorf("unexpected trailing data after CA certificate: %v\n\nCA Certificate:\n%v", string(rest), caPem)
	}

	ca, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse CA certificate: %w\n\tCA Certificate:\n%v", err, caPem)
	}

	return ca, caPem, nil
}

type TCPListener struct {
	Address string     `json:"address"`
	TLS     *TLSConfig `json:"tls,omitempty"`
}

func (t *TCPListener) ToConfig(directory string) (string, error) {
	config := `listener "tcp" {` + "\n"

	if t.Address != "" {
		config += `  address = "` + t.Address + `"` + "\n"
	}

	if t.TLS == nil {
		config += `  tls_disable = true` + "\n"
	} else {
		certPath := filepath.Join(directory, TLS_CERTS_NAME)
		keyPath := filepath.Join(directory, TLS_KEY_NAME)
		if err := t.TLS.Write(certPath, keyPath); err != nil {
			return "", fmt.Errorf("failed to persist TLS configuration: %w", err)
		}
	}

	config += "}\n"
	return config, nil
}

func getConnectionAddr(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("failed to parse address (`%v`): %w", address, err)
	}

	if host == "0.0.0.0" {
		// Connect over localhost instead as we can't connect to an wildcard
		// address as a client.
		host = "127.0.0.1"
	}

	return fmt.Sprintf("%v:%v", host, port), nil
}

func (t *TCPListener) GetConnectAddr() (string, error) {
	return getConnectionAddr(t.Address)
}

type UnixListener struct {
	Path string `json:"path"`
}

func (u *UnixListener) ToConfig(directory string) (string, error) {
	config := `listener "unix" {` + "\n"
	config += `path = "` + u.Path + `"`
	config += "\n}\n"
	return config, nil
}

func (u *UnixListener) GetConnectAddr() (string, error) {
	return "", fmt.Errorf("unix socket cannot be connected to via tcp")
}

type Listener interface {
	ConfigBuilder
	GetConnectAddr() (string, error)
}

var _ Listener = &TCPListener{}
var _ Listener = &UnixListener{}

type RaftStorage struct{}

func (r *RaftStorage) ToConfig(directory string) (string, error) {
	path := filepath.Join(directory, "storage/raft")

	config := `storage "raft" {` + "\n"
	config += `  path = "` + path + `"` + "\n"
	config += "}\n"

	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("failed to make raft storage directory (%v): %w", path, err)
	}

	return config, nil
}

type FileStorage struct{}

func (f *FileStorage) ToConfig(directory string) (string, error) {
	path := filepath.Join(directory, "storage/file")
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("failed to make file storage directory (%v): %w", path, err)
	}

	config := `storage "file" {` + "\n"
	config += `  path = "` + path + `"` + "\n"
	config += "}\n"

	return config, nil
}

type InmemStorage struct{}

func (i *InmemStorage) ToConfig(_ string) (string, error) {
	config := `storage "inmem" {}` + "\n"
	return config, nil
}

type Storage interface {
	ConfigBuilder
}

var _ Storage = &RaftStorage{}
var _ Storage = &FileStorage{}
var _ Storage = &InmemStorage{}

type NodeConfig struct {
	Dev       *DevConfig `json:"dev,omitempty"`
	Listeners []Listener `json:"listners,omitempty"`
	Storage   Storage    `json:"storage,omitempty"`
}

func (n *NodeConfig) Validate() error {
	if len(n.Listeners) == 0 && n.Dev == nil {
		return fmt.Errorf("no listeners specified and dev mode disabled")
	}

	if n.Dev == nil && n.Storage == nil {
		return fmt.Errorf("no storage specified and dev mode disabled")
	}

	return nil
}

func (n *NodeConfig) ToConfig(directory string) (string, error) {
	if err := n.Validate(); err != nil {
		return "", err
	}

	config := ``
	for index, listener := range n.Listeners {
		lConfig, err := listener.ToConfig(directory)
		if err != nil {
			return "", fmt.Errorf("failed to build listener %d (%#v) to config: %w", index, listener, err)
		}

		config += lConfig + "\n"
	}

	// Check if the user has permissions to lock memory and disable otherwise.
	if n.Dev == nil {
		usr, _ := user.Current()
		if usr.Uid != "0" {
			config += "disable_mlock = true\n"
		}
	}

	apiAddr, tls, err := n.GetConnectAddr()
	if err != nil {
		return "", fmt.Errorf("failed to infer connection address: %w\n\tuse a non-raft storage backend or add a tcp listener to this cluster", err)
	}

	scheme := "http"
	if tls {
		scheme += "s"
	}

	if n.Storage != nil {
		sConfig, err := n.Storage.ToConfig(directory)
		if err != nil {
			return "", fmt.Errorf("failed to build storage (%#v) to config: %w", n.Storage, err)
		}

		config += sConfig + "\n"

		if _, ok := n.Storage.(*RaftStorage); ok {
			// Need need to compute the cluster address. This is usually
			// one port higher than the api address.
			host, port, err := net.SplitHostPort(apiAddr)
			if err != nil {
				return "", fmt.Errorf("failed to parse API listen address (%v): %w", apiAddr, err)
			}

			clusterPort, err := strconv.Atoi(port)
			if err != nil {
				return "", fmt.Errorf("failed to parse API listen address port (%v): %w", port, err)
			}

			clusterPort += 1
			clusterAddr := fmt.Sprintf("%s:%d", host, clusterPort)

			config += `cluster_addr = "` + scheme + "://" + clusterAddr + `"` + "\n"
		}
	}

	if n.Dev == nil {
		config += `api_addr = "` + scheme + "://" + apiAddr + `"` + "\n"

		pluginDir := filepath.Join(directory, "plugins")
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create external plugin directory (%v): %w", pluginDir, err)
		}

		config += `plugin_directory = "` + pluginDir + `"` + "\n"
	}

	return config, nil
}

func (n *NodeConfig) GetConnectAddr() (string, bool, error) {
	if err := n.Validate(); err != nil {
		return "", false, err
	}

	if n.Dev != nil {
		address := n.Dev.Address
		if address == "" {
			address = "127.0.0.1:8200"
		}
		host, err := getConnectionAddr(address)
		return host, false, err
	}

	var lastErr error
	for _, listener := range n.Listeners {
		if tcp, ok := listener.(*TCPListener); ok {
			var addr string
			if addr, lastErr = tcp.GetConnectAddr(); lastErr == nil {
				return addr, tcp.TLS != nil, nil
			}
		}
	}

	return "", false, fmt.Errorf("unknown connection address for configuration; last error: %w", lastErr)
}

func (n *NodeConfig) AddArgs(directory string) ([]string, error) {
	args := []string{"server", "-exit-on-core-shutdown"}

	if n.Dev != nil {
		devArgs, err := n.Dev.AddArgs(directory)
		if err != nil {
			return nil, fmt.Errorf("failed to build dev mode args to binary: %w", err)
		}

		args = append(args, devArgs...)
	}

	return args, nil
}

type DevConfig struct {
	Token   string `json:"token,omitempty"`
	Address string `json:"address,omitempty"`
}

func (d *DevConfig) AddArgs(_ string) ([]string, error) {
	args := []string{"-dev"}
	if d.Token != "" {
		args = append(args, fmt.Sprintf("-dev-root-token-id=%s", d.Token))
	}

	if d.Address != "" {
		args = append(args, fmt.Sprintf("-dev-listen-address=%s", d.Address))
	}

	return args, nil
}
