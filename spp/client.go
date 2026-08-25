package spp

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"go.bug.st/serial"
)

// Response represents a response packet received from the controller.
type Response struct {
	CommandCode CommandCode
	StatusCode  StatusCode
	Payload     []byte
}

// Client manages an SPP connection to the Thunderstrike controller.
type Client struct {
	port       serial.Port
	timeout    time.Duration
	totalBytes int
}

// NewClient creates a new SPP client from an established serial connection.
func NewClient(port serial.Port) *Client {
	return &Client{
		port:    port,
		timeout: time.Duration(TimeoutDefaultMs) * time.Millisecond,
	}
}

// SetTimeout sets the response timeout duration.
func (c *Client) SetTimeout(d time.Duration) {
	c.timeout = d
}

// Close closes the underlying serial connection.
func (c *Client) Close() error {
	if c.port != nil {
		return c.port.Close()
	}
	return nil
}

// WriteCommand sends a command packet to the controller.
func (c *Client) WriteCommand(code CommandCode, data []byte) error {
	buf := make([]byte, 3+len(data))
	buf[0] = byte(code)
	binary.LittleEndian.PutUint16(buf[1:3], uint16(len(data)))
	copy(buf[3:], data)

	n, err := c.port.Write(buf)
	if err != nil {
		return fmt.Errorf("spp write %s: %w", code, err)
	}
	if n != len(buf) {
		return fmt.Errorf("spp write %s: short write %d/%d", code, n, len(buf))
	}
	c.totalBytes += n
	return nil
}

// ReadResponse reads a response packet from the controller.
// Loops to skip IN_PROGRESS responses. Enforces an overall deadline.
func (c *Client) ReadResponse() (*Response, error) {
	deadline := time.Now().Add(c.timeout)

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("spp read response: timeout after %v", c.timeout)
		}

		remaining := time.Until(deadline)
		if remaining < 500*time.Millisecond {
			remaining = 500 * time.Millisecond
		}
		c.port.SetReadTimeout(remaining)

		resp, err := c.readOneResponse()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode.IsRetryable() {
			continue
		}
		return resp, nil
	}
}

// readOneResponse reads a single response packet.
func (c *Client) readOneResponse() (*Response, error) {
	header := make([]byte, 3)

	if _, err := io.ReadFull(c.port, header); err != nil {
		return nil, fmt.Errorf("spp read header: %w", err)
	}

	cmdCode := CommandCode(header[0])
	statusCode := StatusCode(header[1])
	payloadLen := int(header[2])

	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(c.port, payload); err != nil {
			return nil, fmt.Errorf("spp read payload (%d bytes): %w", payloadLen, err)
		}
	}

	return &Response{
		CommandCode: cmdCode,
		StatusCode:  statusCode,
		Payload:     payload,
	}, nil
}

// ExecuteCommand sends a command and waits for the matching response.
// Uses ReadResponse which automatically skips IN_PROGRESS status codes.
func (c *Client) ExecuteCommand(code CommandCode, data []byte) (*Response, error) {
	if err := c.WriteCommand(code, data); err != nil {
		return nil, err
	}

	deadline := time.Now().Add(c.timeout)

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("spp execute %s: timeout after %v", code, c.timeout)
		}

		remaining := time.Until(deadline)
		if remaining < 500*time.Millisecond {
			remaining = 500 * time.Millisecond
		}
		c.port.SetReadTimeout(remaining)

		// ReadResponse skips IN_PROGRESS internally.
		resp, err := c.ReadResponse()
		if err != nil {
			return nil, err
		}
		if resp.CommandCode == code {
			return resp, nil
		}
		// Mismatched response code, keep reading (command echo from previous
		// pending operations).
	}
}

// Connect performs the SPP handshake (NOP command).
func (c *Client) Connect() error {
	c.SetTimeout(time.Duration(TimeoutNopMs) * time.Millisecond)

	if err := c.WriteCommand(CmdNop, nil); err != nil {
		return fmt.Errorf("spp connect (NOP wake): %w", err)
	}

	resp, err := c.ExecuteCommand(CmdNop, nil)
	if err != nil {
		return fmt.Errorf("spp connect (NOP handshake): %w", err)
	}

	if !resp.StatusCode.IsOk() {
		return fmt.Errorf("spp connect: handshake failed with status %s", resp.StatusCode)
	}

	return nil
}

// TotalBytes returns the total number of bytes written during firmware flash.
func (c *Client) TotalBytes() int {
	return c.totalBytes
}