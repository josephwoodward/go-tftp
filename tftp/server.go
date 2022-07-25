package tftp

import (
	"fmt"
	"log"
	"net"
)

const (
	DatagramSize = 516 // Maximum supported datagram size
	BlockSize    = DatagramSize - 4
)

type Server struct {
	addr string
}

func NewServer(opts ...Option) *Server {
	c := &ServerOptions{}
	for _, opt := range opts {
		opt(c)
	}

	return &Server{}
}

func (s *Server) ListenAndServer(address string) error {
	conn, err := net.ListenPacket("udp", address)
	if err != nil {
		return err
	}

	defer func() { _ = conn.Close() }()
	log.Printf("Listening on %s ...\n", conn.LocalAddr())

	return s.Serve(conn)

}

func (s *Server) Serve(conn net.PacketConn) error {

	for {
		buf := make([]byte, DatagramSize)

		_, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return err
		}

		fmt.Println(addr)
	}

	return nil
}
