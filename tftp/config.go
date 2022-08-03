package tftp

import "time"

type ServerOptions struct {
	timeout time.Duration
}

type ServerOpt = func(c *ServerOptions)

func WithTimeout(timeout time.Duration) ServerOpt {
	return func(c *ServerOptions) {
		c.timeout = timeout
	}
}
