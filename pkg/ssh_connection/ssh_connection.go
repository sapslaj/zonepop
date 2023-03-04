package ssh_connection

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
		host = host + ":22"
	}
	c := &SSHConnection{
		host: host,
	}

	// TODO: support more configurations
	c.Config = &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", host, c.Config)
	c.Client = client
	if err != nil {
		return nil, fmt.Errorf("ssh_connection: could not connect to host %s: %w", c.host, err)
	}
	return c, nil
}

func (c *SSHConnection) Disconnect() error {
	return c.Client.Close()
}

func (c *SSHConnection) Output(cmd string) ([]byte, error) {
	session, err := c.Client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("ssh_connection: could not start new session to host %s: %w", c.host, err)
	}
	defer session.Close()
	return session.Output(cmd)
}
