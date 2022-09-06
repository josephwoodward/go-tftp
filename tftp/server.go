package tftp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const (
	DatagramSize = 516 // Maximum supported datagram size...
	BlockSize    = DatagramSize - 4
)

type OpCode uint16

//opcode  operation
//1     Read request (RRQ)
//2     Write request (WRQ)
//3     Data (DATA)
//4     Acknowledgment (ACK)
//5     Error (ERROR)
const (
	OpRRQ = iota + 1
	OpWRQ
	OpData
	OpAck
	OpErr
)

type Server struct {
	addr       string
	stop       chan struct{}
	connection net.PacketConn
	opts       *ServerOptions
	wg         sync.WaitGroup
}

// ReadHandler handles server reads
type ReadHandler func(filename string, reader io.Reader) error

// WriteHandler handles write requests
type WriteHandler func(writer io.Writer) error

func NewServer(opts ...ServerOpt) *Server {
	c := &ServerOptions{}
	for _, opt := range opts {
		opt(c)
	}

	return &Server{
		opts: c,
	}
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
	s.stop = make(chan struct{})
	s.connection = conn
	s.connection.SetDeadline(time.Now().Add(s.opts.timeout))

	for {
		select {
		case <-s.stop:
			return nil
		default:
			if err := s.process(); err != nil {
				return err
			}
		}
	}
}

func (s *Server) process() error {
	buf := make([]byte, DatagramSize)
	_, addr, err := s.connection.ReadFrom(buf)
	if err != nil {
		return fmt.Errorf("reading udp packet: %v", err)
	}

	return s.handlePacket(addr, buf)
}

func (s *Server) handlePacket(clientAddr net.Addr, buf []byte) error {
	r := bytes.NewBuffer(buf)

	var code OpCode
	var err error

	// Read the OpCode
	if err = binary.Read(r, binary.BigEndian, &code); err != nil {
		return err
	}

	switch code {
	case OpRRQ:
		rrq := ReadReq{}
		if err = rrq.UnmarshalBinary(buf); err != nil {
			return err
		}

		log.Printf("[%s] requested file : %s", clientAddr, rrq.Filename)

		conn, err := net.Dial("udp", clientAddr.String())
		if err != nil {
			return err
		}

		s.wg.Add(1)
		go func() {
			// TODO: We've read the request, now to send it back...
			s.opts.readHandler(rrq.Filename)
		}()

		defer func() {
			_ = conn.Close()
		}()
	}

	return nil
}
