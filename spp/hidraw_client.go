// Package spp — HID-over-SPP transport.
//
// The Thunderstrike controller uses the same HID report protocol over both
// USB and Bluetooth SPP connections. When connected via Bluetooth, the
// Bt_Ops.sendNextRequest() method calls SerialDeviceProtocol.SdpHandle.
// device_write() to send raw HID report bytes over the RFCOMM serial link.
//
// Report format (identical to USB HID):
//   Request:  [0x04] [cmd_ordinal] [transaction_id] [data...] [zero-padded to ReportLen]
//   Response: [0x03] [cmd_ordinal] [transaction_id] [response_data...]
//
// This file implements HidrawClient which satisfies the hid.DeviceInterfacer
// interface, allowing the existing hid.QueryInfo() and command parsers to be
// reused for Bluetooth-connected devices without modification.
package spp

import (
	"fmt"
	"time"

	"go.bug.st/serial"
	"thunderstrike-controller-tool/hid"
)

// ReportLen is the HID report length for SPP transport.
// From Bt_Ops.smali: mReportLen = getHidrawReportLength() + 3 = 30 + 3 = 33.
const ReportLen = 33

// DefaultReadTimeout is the default timeout for reading a response over SPP.
const DefaultReadTimeout = 5 * time.Second

// HidrawClient communicates with the controller over Bluetooth SPP using
// the same HID report protocol as USB connections.
type HidrawClient struct {
	port          serial.Port
	transactionID byte
}

// NewHidrawClient opens a Bluetooth SPP serial port and returns a client
// that implements hid.DeviceInterfacer. The serial port is configured with
// the standard SPP baud rate and no flow control.
func NewHidrawClient(comPort string) (*HidrawClient, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(comPort, mode)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", comPort, err)
	}

	return &HidrawClient{port: port}, nil
}

// ProbeController tries to connect to a COM port and send a NOP command.
// Returns true if the device responds (i.e. it's a Thunderstrike controller).
// Uses a short timeout so it doesn't block on non-controller ports.
func ProbeController(comPort string) bool {
	client, err := NewHidrawClient(comPort)
	if err != nil {
		return false
	}
	defer client.Close()

	// Use a short 2-second timeout for probing.
	client.port.SetReadTimeout(2 * time.Second)

	// Send NOP (0x00) — the controller should respond with [0x03][0x00][txn][data].
	_, err = client.SendCommand(hid.CmdNop, nil)
	return err == nil
}

// SendCommand sends a HID report command over SPP and reads the matching
// response. It mirrors the USB HID SendCommand logic:
//  1. Build a 33-byte request report [0x04][cmd][txn][data][pad]
//  2. Write the report bytes to the serial port
//  3. Read response bytes until we get a matching [0x03][cmd][txn] header
//
// If the device responds with 0xFF as the command ordinal, it means
// CMD_ERROR — the command is not supported by the firmware.
func (c *HidrawClient) SendCommand(cmd hid.BtHidrawCommand, data []byte) ([]byte, error) {
	report := c.buildRequestReport(cmd, data)
	txnID := report[2]

	// Purge any stale data in the serial receive buffer.
	c.purgeRxBuffer()

	// Send the request.
	if _, err := c.port.Write(report); err != nil {
		return nil, fmt.Errorf("%s: spp write: %w", cmd, err)
	}

	// Read the matching response.
	respData, err := c.readMatchingResponse(txnID, cmd, DefaultReadTimeout)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", cmd, err)
	}

	return respData, nil
}

// Close closes the underlying serial port.
func (c *HidrawClient) Close() error {
	if c.port != nil {
		return c.port.Close()
	}
	return nil
}

// buildRequestReport constructs the 33-byte request report buffer.
func (c *HidrawClient) buildRequestReport(cmd hid.BtHidrawCommand, data []byte) []byte {
	buf := make([]byte, ReportLen)
	buf[0] = hid.RequestPrefix // 0x04
	buf[1] = byte(cmd)
	buf[2] = c.nextTransactionID()

	maxDataLen := ReportLen - 3
	dataLen := len(data)
	if dataLen > maxDataLen {
		dataLen = maxDataLen
	}
	if dataLen > 0 {
		copy(buf[3:3+dataLen], data[:dataLen])
	}
	return buf
}

// nextTransactionID returns the next incrementing transaction ID byte.
func (c *HidrawClient) nextTransactionID() byte {
	c.transactionID++
	return c.transactionID
}

// purgeRxBuffer drains any pending data in the serial receive buffer.
// This prevents stale responses from previous commands being misread.
func (c *HidrawClient) purgeRxBuffer() {
	buf := make([]byte, 256)
	for {
		// Non-blocking read: 1ms timeout to flush available data.
		c.port.SetReadTimeout(1 * time.Millisecond)
		n, err := c.port.Read(buf)
		if err != nil || n == 0 {
			break
		}
	}
	c.port.SetReadTimeout(DefaultReadTimeout)
}

// readMatchingResponse reads bytes from the serial port until it finds a
// response report matching the expected transaction ID and command.
//
// The response format is: [0x03] [cmd_ordinal] [transaction_id] [data...]
//
// If the command ordinal is 0xFF (CMD_ERROR) with matching txn ID, the
// command is not supported.
//
// Important: go.bug.st/serial returns an error on read timeout. We must
// continue retrying until the overall deadline is reached, rather than
// aborting on the first timeout error.
func (c *HidrawClient) readMatchingResponse(expectedTxn byte, expectedCmd hid.BtHidrawCommand, timeout time.Duration) ([]byte, error) {
	// Use a short per-read timeout so we can loop and check the deadline.
	// The overall deadline controls how long we wait in total.
	perReadTimeout := 500 * time.Millisecond
	deadline := time.Now().Add(timeout)
	c.port.SetReadTimeout(perReadTimeout)
	defer c.port.SetReadTimeout(DefaultReadTimeout)

	// Accumulate received bytes in a buffer.
	var rxBuf []byte

	for time.Now().Before(deadline) {
		// Read available bytes.
		tmp := make([]byte, hid.MaxResponseSize)
		n, err := c.port.Read(tmp)
		if err != nil {
			// On timeout/error, just continue the loop.
			// The deadline check will eventually break us out.
			continue
		}
		if n == 0 {
			continue
		}

		rxBuf = append(rxBuf, tmp[:n]...)

		// Try to parse a complete response from the accumulated buffer.
		// Response format: [0x03] [cmd_ordinal] [txn_id] [data...]
		for len(rxBuf) >= 3 {
			// Find response prefix 0x03.
			idx := -1
			for i, b := range rxBuf {
				if b == hid.ResponsePrefix {
					idx = i
					break
				}
			}
			if idx < 0 {
				// No prefix found — discard all data.
				rxBuf = rxBuf[:0]
				break
			}
			// Discard bytes before the prefix.
			rxBuf = rxBuf[idx:]

			if len(rxBuf) < 3 {
				// Need at least 3 bytes for the header.
				break
			}

			respCmd := hid.BtHidrawCommand(rxBuf[1])
			respTxn := rxBuf[2]

			// The response data is everything after the 3-byte header.
			// We don't know the exact length, so take what we have.
			// If more data is arriving, it will be appended in the next read.
			respData := rxBuf[3:]

			// Check for CMD_ERROR (0xFF) with matching txn ID.
			if byte(respCmd) == 0xFF && respTxn == expectedTxn {
				return nil, hid.ErrCmdError
			}

			// Check for a matching response.
			if respCmd == expectedCmd && respTxn == expectedTxn {
				// Return a copy of the response data.
				result := make([]byte, len(respData))
				copy(result, respData)
				return result, nil
			}

			// Mismatch — discard the 3-byte header and keep scanning.
			rxBuf = rxBuf[3:]
		}
	}

	return nil, fmt.Errorf("timeout waiting for %s response (txn=%d)", expectedCmd, expectedTxn)
}
