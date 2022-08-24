package tftp

import "time"

type ServerOptions struct {
	timeout      time.Duration
	readHandler  ReadHandler
	writeHandler WriteHandler
}

type ServerOpt = func(c *ServerOptions)

// WithTimeout sets the deadline for both read and write connections
func WithTimeout(timeout time.Duration) ServerOpt {
	return func(c *ServerOptions) {
		c.timeout = timeout
	}
}

func WithReader(reader ReadHandler) ServerOpt {
	return func(c *ServerOptions) {
		c.readHandler = reader
	}
}
