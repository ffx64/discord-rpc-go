//go:build windows

package ipc

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/Microsoft/go-winio"
	"github.com/google/uuid"
)

type Conn struct {
	conn   net.Conn
	mu     sync.Mutex
	closed bool
	reader *bufio.Reader
}

func NewConn(appID string) (*Conn, error) {
	c, err := winio.DialPipe(ipcPath(), nil)
	if err != nil {
		return nil, err
	}

	return &Conn{
		conn:   c,
		reader: bufio.NewReader(c),
	}, nil
}

func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Conn) SendRaw(b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("connection closed")
	}
	_, err := c.conn.Write(b)
	return err
}

func (c *Conn) SendOp(op OpCode, payload any) error {
	data, err := EncodeFrameOp(op, payload)
	if err != nil {
		return err
	}
	return c.SendRaw(data)
}

func (c *Conn) Receive() (OpCode, json.RawMessage, error) {
	// read 8 bytes header
	header := make([]byte, 8)
	if _, err := c.reader.Read(header); err != nil {
		return 0, nil, err
	}
	op := OpCode(binary.LittleEndian.Uint32(header[0:4]))
	length := int(binary.LittleEndian.Uint32(header[4:8]))
	if length < 0 || length > 10*1024*1024 {
		return 0, nil, fmt.Errorf("invalid payload length %d", length)
	}
	payload := make([]byte, length)
	if _, err := c.reader.Read(payload); err != nil {
		return 0, nil, err
	}
	return op, json.RawMessage(payload), nil
}

func ipcPath() string {
	return `\\.\pipe\discord-ipc-0`
}

func BuildDispatchPayload(cmd string, args any) (map[string]any, error) {
	return map[string]any{
		"cmd":   cmd,
		"args":  args,
		"nonce": uuid.New().String(),
	}, nil
}

func ActivityArgsWithPid(activity any) map[string]any {
	return map[string]any{
		"pid":      os.Getpid(),
		"activity": activity,
	}
}
