// Package rcon is a minimal client for the Source RCON protocol used by
// Minecraft servers, just enough to authenticate and run one command per
// connection.
package rcon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	packetAuth        int32 = 3
	packetExecCommand int32 = 2

	authFailedID = -1

	defaultTimeout = 10 * time.Second
)

var ErrAuthFailed = errors.New("rcon: authentication failed (wrong password?)")

type Client struct {
	conn  net.Conn
	reqID int32
}

// Dial connects to a Minecraft RCON server at addr and authenticates with
// password.
func Dial(addr, password string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, defaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("rcon: connecting to %s: %w", addr, err)
	}

	c := &Client{conn: conn}
	if err := c.authenticate(password); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Execute runs a single console command and returns its text response.
func (c *Client) Execute(command string) (string, error) {
	if err := c.send(packetExecCommand, command); err != nil {
		return "", err
	}
	_, body, err := c.receive()
	return body, err
}

func (c *Client) authenticate(password string) error {
	if err := c.send(packetAuth, password); err != nil {
		return err
	}
	id, _, err := c.receive()
	if err != nil {
		return err
	}
	if id == authFailedID {
		return ErrAuthFailed
	}
	return nil
}

func (c *Client) send(packetType int32, payload string) error {
	c.reqID++

	body := new(bytes.Buffer)
	binary.Write(body, binary.LittleEndian, c.reqID)
	binary.Write(body, binary.LittleEndian, packetType)
	body.WriteString(payload)
	body.Write([]byte{0, 0})

	c.conn.SetWriteDeadline(time.Now().Add(defaultTimeout))
	if err := binary.Write(c.conn, binary.LittleEndian, int32(body.Len())); err != nil {
		return fmt.Errorf("rcon: writing packet length: %w", err)
	}
	if _, err := c.conn.Write(body.Bytes()); err != nil {
		return fmt.Errorf("rcon: writing packet body: %w", err)
	}
	return nil
}

func (c *Client) receive() (id int32, payload string, err error) {
	c.conn.SetReadDeadline(time.Now().Add(defaultTimeout))

	var length int32
	if err := binary.Read(c.conn, binary.LittleEndian, &length); err != nil {
		return 0, "", fmt.Errorf("rcon: reading packet length: %w", err)
	}
	if length < 10 {
		return 0, "", fmt.Errorf("rcon: malformed packet (length %d)", length)
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(c.conn, buf); err != nil {
		return 0, "", fmt.Errorf("rcon: reading packet body: %w", err)
	}

	id = int32(binary.LittleEndian.Uint32(buf[0:4]))
	// buf[4:8] is the packet type, which callers here don't need.
	payload = string(buf[8 : len(buf)-2])
	return id, payload, nil
}
