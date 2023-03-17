package sshconnection

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

type SSHConnection struct {
	host   string
	Config *ssh.ClientConfig
	Client *ssh.Client
}

func Connect(host, username, password string) (*SSHConnection, error) {
	if !strings.Contains(host, ":") {
		host += ":22"
	}
	c := &SSHConnection{
		host: host,
	}

	// TODO: support more configurations
	c.Config = &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: make this not harcoded
	}
	client, err := ssh.Dial("tcp", host, c.Config)
	c.Client = client
	if err != nil {
		return nil, fmt.Errorf("sshconnection: could not connect to host %s: %w", c.host, err)
	}
	return c, nil
}

func (c *SSHConnection) Disconnect() error {
	return c.Client.Close()
}

func (c *SSHConnection) Output(cmd string) ([]byte, error) {
	session, err := c.Client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("sshconnection: could not start new session to host %s: %w", c.host, err)
	}
	defer session.Close()
	return session.Output(cmd)
}
