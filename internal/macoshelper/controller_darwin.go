//go:build darwin

package macoshelper

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var ErrUnavailable = errors.New("macOS helper unavailable")

type Controller struct {
	helperPath string

	mu           sync.Mutex
	cmd          *exec.Cmd
	listener     net.Listener
	conn         net.Conn
	encoder      *json.Encoder
	tempDir      string
	last         *Message
	lastSettings *Message
	closed       bool

	commands chan string
	errors   chan error
	done     chan struct{}
}

func New(helperPath string) *Controller {
	return &Controller{
		helperPath: strings.TrimSpace(helperPath),
		commands:   make(chan string, 8),
		errors:     make(chan error, 4),
		done:       make(chan struct{}),
	}
}

func (c *Controller) Start() error {
	helperPath, err := resolveHelperPath(c.helperPath)
	if err != nil {
		return err
	}

	dir, err := os.MkdirTemp("", "sonosh-macos-helper-*")
	if err != nil {
		return err
	}
	socketPath := filepath.Join(dir, "helper.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		_ = os.RemoveAll(dir)
		return err
	}

	cmd := exec.Command(helperPath, "--socket", socketPath) //nolint:gosec // helper path is explicit/env/sibling executable.
	if err := cmd.Start(); err != nil {
		_ = listener.Close()
		_ = os.RemoveAll(dir)
		return err
	}

	c.mu.Lock()
	c.cmd = cmd
	c.listener = listener
	c.tempDir = dir
	c.mu.Unlock()

	go c.acceptLoop(listener)
	go c.waitProcess(cmd)
	return nil
}

func (c *Controller) Commands() <-chan string {
	return c.commands
}

func (c *Controller) Errors() <-chan error {
	return c.errors
}

func (c *Controller) Publish(msg Message) {
	c.sendOrStore(msg)
}

func (c *Controller) Clear() {
	c.sendOrStore(Message{Type: "clear"})
}

func (c *Controller) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	conn := c.conn
	listener := c.listener
	cmd := c.cmd
	dir := c.tempDir
	c.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
	if listener != nil {
		_ = listener.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}
	if dir != "" {
		_ = os.RemoveAll(dir)
	}
	close(c.done)
	return nil
}

func (c *Controller) acceptLoop(listener net.Listener) {
	conn, err := listener.Accept()
	if err != nil {
		c.sendError(err)
		return
	}

	c.mu.Lock()
	c.conn = conn
	c.encoder = json.NewEncoder(conn)
	last := c.last
	lastSettings := c.lastSettings
	c.mu.Unlock()

	_ = c.send(HelloMessage())
	if lastSettings != nil {
		_ = c.send(*lastSettings)
	}
	if last != nil {
		_ = c.send(*last)
	}
	c.readLoop(conn)
}

func (c *Controller) readLoop(conn net.Conn) {
	dec := json.NewDecoder(conn)
	for {
		var msg Message
		if err := dec.Decode(&msg); err != nil {
			c.sendError(err)
			return
		}
		switch msg.Type {
		case "command":
			command := strings.TrimSpace(msg.Command)
			if command != "" {
				c.sendCommand(command)
			}
		case "error":
			if msg.Text != "" {
				c.sendError(errors.New(msg.Text))
			}
		}
	}
}

func (c *Controller) waitProcess(cmd *exec.Cmd) {
	err := cmd.Wait()
	if err != nil {
		c.sendError(fmt.Errorf("macOS helper exited: %w", err))
	}
}

func (c *Controller) sendOrStore(msg Message) {
	c.mu.Lock()
	if msg.Type == "settings" {
		c.lastSettings = &msg
	} else {
		c.last = &msg
	}
	encoder := c.encoder
	c.mu.Unlock()
	if encoder != nil {
		_ = c.send(msg)
	}
}

func (c *Controller) send(msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.encoder == nil {
		return nil
	}
	return c.encoder.Encode(msg)
}

func (c *Controller) sendCommand(command string) {
	select {
	case c.commands <- command:
	case <-c.done:
	}
}

func (c *Controller) sendError(err error) {
	if err == nil || errors.Is(err, net.ErrClosed) {
		return
	}
	select {
	case c.errors <- err:
	case <-c.done:
	default:
	}
}

func resolveHelperPath(explicit string) (string, error) {
	if explicit != "" {
		if info, err := os.Stat(explicit); err == nil && !info.IsDir() {
			return explicit, nil
		}
		return "", fmt.Errorf("%w: %s", ErrUnavailable, explicit)
	}
	if env := strings.TrimSpace(os.Getenv("SONOSH_MAC_HELPER")); env != "" {
		if info, err := os.Stat(env); err == nil && !info.IsDir() {
			return env, nil
		}
		return "", fmt.Errorf("%w: %s", ErrUnavailable, env)
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	for _, candidate := range []string{
		filepath.Join(filepath.Dir(exe), "sonosh-macos-helper"),
		filepath.Join("helpers", "macos", "sonosh-helper", ".build", "release", "sonosh-macos-helper"),
		filepath.Join("helpers", "macos", "sonosh-helper", ".build", "debug", "sonosh-macos-helper"),
	} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", ErrUnavailable
}
