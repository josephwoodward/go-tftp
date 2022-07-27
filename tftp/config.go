package tftp

import "time"

type ServerOptions struct {
	timeout time.Duration
}

type Option = func(c *ServerOptions)

func WithTimeout(timeout time.Duration) Option {
	return func(c *ServerOptions) {
		c.timeout = timeout
	}
}
