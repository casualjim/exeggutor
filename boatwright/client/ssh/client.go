package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/reverb/exeggutor/boatwright"

	gossh "code.google.com/p/go.crypto/ssh"
	"code.google.com/p/go.crypto/ssh/agent"
)

var (
	newLine = []byte("\n")
)

type printWriter struct {
	writer io.Writer
	item   boatwright.InventoryItem
}

func (w *printWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	lines := strings.Split(s, "\n")
	for _, line := range lines[0 : len(lines)-1] {
		_, err = fmt.Printf("%s[%s][%s]: %s\n", w.item.Cluster, w.item.Name, w.item.PublicHost, line)
		if err != nil {
			return
		}
	}
	n = len(p)
	return
}

type channelWriter struct {
	channel chan<- boatwright.RemoteEvent
	item    boatwright.InventoryItem
}

func (c *channelWriter) Write(p []byte) (n int, err error) {
	lines := bytes.Split(p, newLine)
	for _, line := range lines[0 : len(lines)-1] {
		c.channel <- boatwright.RemoteEvent{Host: c.item, Line: line}
	}
	n = len(p)
	return
}

// SshClient connects to a remote host to execute a command
type SshClient struct {
	conn      *gossh.Client
	config    *boatwright.Config
	host      boatwright.InventoryItem
	connected bool
}

// New create a new ssh client
func New(config *boatwright.Config) *SshClient {
	return &SshClient{config: config, connected: false}
}

// Connect create the connection session
func (s *SshClient) Connect(item boatwright.InventoryItem) error {
	fmt.Printf("Connecting to %s in %s at %s with user %s and key %s\n", item.Name, item.Cluster, item.PublicHost, s.config.SSH.User, s.config.SSH.KeyFile)
	agConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		log.Fatal(err)
	}
	defer agConn.Close()
	ag := agent.NewClient(agConn)
	auths := []gossh.AuthMethod{gossh.PublicKeysCallback(ag.Signers)}

	clientConfig := &gossh.ClientConfig{
		User: s.config.SSH.User,
		Auth: auths,
	}
	conn, err := gossh.Dial("tcp", item.PublicHost+":22", clientConfig)
	if err != nil {
		return err
	}
	s.conn = conn
	s.host = item
	s.connected = true
	return nil
}

// RunSimple runs a command an attaches stdin,stdout and stderr to the session.
func (s *SshClient) RunSimple(cmd string) error {
	if !s.connected {
		return errors.New("You need to connect the ssh client firest")
	}
	session, err := s.conn.NewSession()
	if err != nil {
		return err
	}
	session.Stdout = &printWriter{writer: os.Stdout, item: s.host}
	session.Stderr = &printWriter{writer: os.Stderr, item: s.host}
	session.Stdin = os.Stdin
	return session.Run(cmd)
}

// RunStreaming runs a command but writes every received line as an event to a channel
func (s *SshClient) RunStreaming(cmd string) (chan boatwright.RemoteEvent, error) {
	if !s.connected {
		return nil, errors.New("You need to connect the ssh client first")
	}
	hatch := make(chan boatwright.RemoteEvent, 1000)
	go func() {
		writer := &channelWriter{item: s.host, channel: hatch}
		session, err := s.conn.NewSession()
		if err != nil {
			fmt.Errorf("Failed to establish ssh session, because: %v", err)
			return
		}
		session.Stdout = writer
		session.Stderr = writer
		err = session.Run(cmd)
		if err != nil {
			fmt.Errorf("Command %s failed on %s[%s][%s], because %v", cmd, s.host.Cluster, s.host.Name, s.host.PublicHost, err)
		}
	}()
	return hatch, nil
}

// Disconnect disconnects this ssh client
func (s *SshClient) Disconnect() error {
	if s.connected {
		s.connected = false
		return s.conn.Close()
	}
	return nil
}
