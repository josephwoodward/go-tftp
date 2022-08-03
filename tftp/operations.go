package tftp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

type ErrCode uint16

const (
	ErrUnknown ErrCode = iota
	ErrNotFound
	ErrAccessViolation
	ErrDiskFull
	ErrIllegalOp
	ErrUnknownID
	ErrFileExists
	ErrNoUser
)

// ReadReq acts as the initial read request packet (RRQ) informing the server which file it would like to read
//2 bytes     string    1 byte     string   1 byte
//------------------------------------------------
//| Opcode |  Filename  |   0  |    Mode    |   0  |
//------------------------------------------------
type ReadReq struct {
	Filename string
	Mode     string
	Options  Option
}

// Option TFTP blksize. On default the TFTP protocol intends data transmission by 512 bytes blocks. Regarding the fact that present local networks MTU is usually equal to 1500 bytes or more, this block size is not effective. TFTP blksize option let change the block size, improving data transmission effectiveness.
// TFTP tsize. With help of the TFTP tsize option the client can request from the server the size of the file being transmitted.
// TFTP timeout. TFTP timeout option allows the client to set server timeout for the file being transmitted.
// TFTP multicast. This option enables multicast file transmission mode.
type Option map[string]string

// MarshalBinary won't work yet as we're only focusing on downloading
func (q *ReadReq) MarshalBinary() ([]byte, error) {
	mode := "octet"
	if q.Mode != "" {
		mode = q.Mode
	}

	// capacity: operation code + filename + 0 byte + mode + 0 byte
	// https://datatracker.ietf.org/doc/html/rfc1350#section-5
	capacity := 2 + 2 + len(q.Filename) + 1 + len(q.Mode) + 1

	b := new(bytes.Buffer)
	b.Grow(capacity)

	// Write Opcode
	if err := binary.Write(b, binary.BigEndian, OpRRQ); err != nil {
		return nil, err
	}

	// Write Filename
	if _, err := b.WriteString(q.Filename); err != nil {
		return nil, err
	}

	// Write null byte
	if err := b.WriteByte(0); err != nil {
		return nil, err
	}

	// Write Mode
	if _, err := b.WriteString(mode); err != nil {
		return nil, err
	}

	// Write another null byte
	if err := b.WriteByte(0); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (q *ReadReq) UnmarshalBinary(p []byte) error {
	s := bytes.Split(p[2:], []byte{0})
	if len(s) < 2 {
		return fmt.Errorf("missing filename or mode")
	}

	// Filename, mode
	q.Filename = string(s[0])
	q.Mode = string(s[1])
	if len(s) < 4 {
		return nil
	}

	// Options
	q.Options = make(Option)
	for i := 2; i+1 < len(s); i += 2 {
		q.Options[string(s[i])] = string(s[i+1])
	}

	return nil
}

// Data acts as the data packet that will transfer the files payload
// 2 bytes     2 bytes      n bytes
// ----------------------------------
// | Opcode |   Block #  |   Data   |
// ----------------------------------
type Data struct {
	// Block enables UDP reliability by incrementing on each packet sent,
	// the client discriminate between new packets and duplicates, sending an ack including the block number to
	// confirm delivery
	Block   uint16
	Payload io.Reader
}

func (d *Data) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	b.Grow(DatagramSize)

	d.Block++

	if err := binary.Write(b, binary.BigEndian, uint16(OpData)); err != nil {
		return nil, err
	}

	if err := binary.Write(b, binary.BigEndian, d.Block); err != nil { // write block number to packet
		return nil, err
	}

	// Every packet will be BlockSize (516 bytes) expect for the last one, which is how the client knows
	// it's reached the end of the stream
	_, err := io.CopyN(b, d.Payload, BlockSize)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return b.Bytes(), nil
}

func (d *Data) UnmarshalBinary(p []byte) error {
	// Sanity check the payload data
	if l := len(p); l < 4 || l > DatagramSize {
		return errors.New("invalid DATA")
	}

	var opcode any
	// Read opcode from packet
	err := binary.Read(bytes.NewReader(p[:2]), binary.BigEndian, &opcode)
	if err != nil || opcode != OpData {
		return errors.New("invalid DATA")
	}

	// Read block number
	err = binary.Read(bytes.NewReader(p[2:4]), binary.BigEndian, &d.Block)
	if err != nil {
		return errors.New("invalid DATA")
	}

	// Read byte slice to get the end to get data
	d.Payload = bytes.NewBuffer(p[4:])

	return nil
}

// Ack responds to the server with a block number to inform the server
// which packet it just received
// 2 bytes     2 bytes
// ---------------------
// | Opcode |   Block #  |
// ---------------------
type Ack uint16

func (a *Ack) MarshalBinary() ([]byte, error) {
	capacity := 2 + 2 // operation code + block number

	b := new(bytes.Buffer)
	b.Grow(capacity)

	err := binary.Write(b, binary.BigEndian, OpAck) // Write ack op code to buffer
	if err != nil {
		return nil, err
	}

	err = binary.Write(b, binary.BigEndian, &a) // Now write block number
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (a *Ack) UnmarshalBinary(p []byte) error {
	var code OpCode

	r := bytes.NewReader(p)

	if err := binary.Read(r, binary.BigEndian, &code); err != nil {
		return err
	}

	if code != OpAck {
		return errors.New("invalid ACK")
	}

	return binary.Read(r, binary.BigEndian, a)
}

// Err packet
// 2 bytes     2 bytes       string    1 byte
// -----------------------------------------
// | Opcode |  ErrorCode |   ErrMsg   |   0  |
// -----------------------------------------
type Err struct {
	Error   ErrCode
	Message string
}

func (e Err) MarshalBinary() ([]byte, error) {
	capacity := 2 + 2 + len(e.Message) + 1

	b := new(bytes.Buffer)
	b.Grow(capacity)

	err := binary.Write(b, binary.BigEndian, OpErr) // Write OpErr op code to buffer
	if err != nil {
		return nil, err
	}

	// Now write error code
	if err = binary.Write(b, binary.BigEndian, e.Error); err != nil {
		return nil, err
	}

	_, err = b.WriteString(e.Message)
	if err != nil {
		return nil, err
	}

	if err = b.WriteByte(0); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (e Err) UnmarshalBinary(p []byte) error {
	r := bytes.NewBuffer(p)

	var code OpCode

	if err := binary.Read(r, binary.BigEndian, &code); err != nil { // read op code
		return err
	}

	if code != OpErr {
		return errors.New("invalid ERROR")
	}

	if err := binary.Read(r, binary.BigEndian, &e.Error); err != nil {
		return err
	}

	var err error
	e.Message, err = r.ReadString(0)
	e.Message = strings.TrimRight(e.Message, "\x00") // remove the 0-byte

	return err
}
