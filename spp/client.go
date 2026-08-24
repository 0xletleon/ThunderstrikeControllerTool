package spp

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// Response represents a response packet received from the controller.
type Response struct {
	CommandCode  CommandCode
	StatusCode   StatusCode
	Payload      []byte
}

// Client manages an SPP connection to the Thunderstrike controller.
// It provides methods for sending commands and reading responses
// over a Bluetooth SPP (RFCOMM) serial connection.
type Client struct {
	port       io.ReadWriteCloser
	timeout    time.Duration
	totalBytes int
}

// NewClient creates a new SPP client from an established serial connection.
// The port parameter should be an open serial port connected to the
// controller's SPP service.
func NewClient(port io.ReadWriteCloser) *Client {
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
// The packet format is: [CommandCode] [2-byte LE length] [data...]
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
	return nil
}

// ReadResponse reads a response packet from the controller.
// It loops to skip STATUS_CMD_IN_PROGRESS responses.
// Packet format: [CommandCode] [StatusCode] [payload_len] [payload...]
func (c *Client) ReadResponse() (*Response, error) {
	for {
		resp, err := c.readOneResponse()
		if err != nil {
			return nil, err
		}
		// Skip "in progress" responses and wait for final status.
		if resp.StatusCode.IsRetryable() {
			continue
		}
		return resp, nil
	}
}

// readOneResponse reads a single response packet.
func (c *Client) readOneResponse() (*Response, error) {
	header := make([]byte, 3)

	// Read 3-byte header: [cmd_code] [status_code] [payload_len]
	if _, err := io.ReadFull(c.port, header); err != nil {
		return nil, fmt.Errorf("spp read header: %w", err)
	}

	cmdCode := CommandCode(header[0])
	statusCode := StatusCode(header[1])
	payloadLen := int(header[2])

	// Read payload
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
// It retries reading until the response's CommandCode matches the sent command.
func (c *Client) ExecuteCommand(code CommandCode, data []byte) (*Response, error) {
	if err := c.WriteCommand(code, data); err != nil {
		return nil, err
	}

	// Read responses until we get one matching our command code.
	for {
		resp, err := c.ReadResponse()
		if err != nil {
			return nil, err
		}
		if resp.CommandCode == code {
			return resp, nil
		}
		// Mismatched response code, keep reading.
	}
}

// Connect performs the SPP handshake (NOP command).
// Must be called after establishing the serial connection.
//
// The original app sends NOP twice:
//  1. WriteCommand(NOP) — wake up the controller
//  2. ExecuteCommand(NOP) — confirm handshake (WriteCommand + ReadResponse)
//
// Timeout is 5000ms (0x1388 in smali).
func (c *Client) Connect() error {
	c.SetTimeout(time.Duration(TimeoutNopMs) * time.Millisecond)

	// Step 1: Wake up the controller with a bare WriteCommand (no response read).
	if err := c.WriteCommand(CmdNop, nil); err != nil {
		return fmt.Errorf("spp connect (NOP wake): %w", err)
	}

	// Step 2: ExecuteCommand (WriteCommand + ReadResponse) to confirm handshake.
	resp, err := c.ExecuteCommand(CmdNop, nil)
	if err != nil {
		return fmt.Errorf("spp connect (NOP handshake): %w", err)
	}

	if !resp.StatusCode.IsOk() {
		return fmt.Errorf("spp connect: handshake failed with status %s",
			resp.StatusCode)
	}

	return nil
}

// TotalBytes returns the total number of bytes written during firmware flash.
func (c *Client) TotalBytes() int {
	return c.totalBytes
}
