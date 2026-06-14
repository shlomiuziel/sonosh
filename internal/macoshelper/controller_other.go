//go:build !darwin

package macoshelper

import "errors"

var ErrUnavailable = errors.New("macOS helper unavailable")

type Controller struct {
	commands chan string
	errors   chan error
}

func New(string) *Controller {
	return &Controller{
		commands: make(chan string),
		errors:   make(chan error),
	}
}

func (c *Controller) Start() error {
	return ErrUnavailable
}

func (c *Controller) Commands() <-chan string {
	return c.commands
}

func (c *Controller) Errors() <-chan error {
	return c.errors
}

func (*Controller) Publish(Message) {}

func (*Controller) Clear() {}

func (*Controller) Close() error {
	return nil
}
