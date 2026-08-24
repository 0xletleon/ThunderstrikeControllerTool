// debug_probe — test SPP flash protocol NOP response data.
// Usage: go run ./cmd/debug_probe/ COM3
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"go.bug.st/serial"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug_probe <COM_PORT>")
		os.Exit(1)
	}
	portName := os.Args[1]

	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	fmt.Printf("Opening %s...\n", portName)
	port, err := serial.Open(portName, mode)
	if err != nil {
		fmt.Printf("Open failed: %v\n", err)
		os.Exit(1)
	}
	defer port.Close()
	fmt.Printf("Opened successfully.\n\n")

	// SPP flash protocol: [cmd][2B LE len][data]
	// NOP = 0x00, no data → [0x00, 0x00, 0x00]
	// First NOP: wake up (write only, don't read)
	nopWake := []byte{0x00, 0x00, 0x00}
	port.SetReadTimeout(2 * time.Second)
	_, err = port.Write(nopWake)
	if err != nil {
		fmt.Printf("NOP wake write failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Sent NOP wake (no read)")

	// Small delay
	time.Sleep(100 * time.Millisecond)

	// Drain any data
	buf := make([]byte, 256)
	port.SetReadTimeout(100 * time.Millisecond)
	for {
		n, _ := port.Read(buf)
		if n == 0 {
			break
		}
		fmt.Printf("  Drained %d bytes: % X\n", n, buf[:n])
	}

	// Second NOP: ExecuteCommand (write + read response)
	fmt.Println("\nSending NOP handshake (write + read)...")
	nopCmd := []byte{0x00, 0x00, 0x00}
	port.SetReadTimeout(5 * time.Second)
	n, err := port.Write(nopCmd)
	if err != nil {
		fmt.Printf("NOP write failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Sent %d bytes: % X\n", n, nopCmd)

	// Read response header first (3 bytes)
	respBuf := make([]byte, 256)
	port.SetReadTimeout(5 * time.Second)
	n, err = port.Read(respBuf)
	if err != nil {
		fmt.Printf("Read header failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Received %d bytes: % X\n", n, respBuf[:n])

	if n >= 3 {
		cmd := respBuf[0]
		status := respBuf[1]
		payloadLen := respBuf[2]
		fmt.Printf("\n  cmd=%d status=%d payloadLen=%d\n", cmd, status, payloadLen)

		// Read payload if not yet fully received
		totalRead := n
		if payloadLen > 0 && totalRead < 3+int(payloadLen) {
			fmt.Printf("  Need %d more payload bytes, reading...\n", 3+int(payloadLen)-totalRead)
			port.SetReadTimeout(2 * time.Second)
			for totalRead < 3+int(payloadLen) {
				n2, err2 := port.Read(respBuf[totalRead:])
				if err2 != nil {
					fmt.Printf("  Read payload failed: %v\n", err2)
					break
				}
				if n2 == 0 {
					break
				}
				totalRead += n2
				fmt.Printf("  Read %d more bytes (total: %d)\n", n2, totalRead)
			}
		}

		if totalRead >= 3+int(payloadLen) && payloadLen > 0 {
			payload := respBuf[3 : 3+int(payloadLen)]
			fmt.Printf("  payload (%d bytes): % X\n", len(payload), payload)

			// Try to interpret payload
			fmt.Println("\n  Payload interpretation:")
			if len(payload) >= 2 {
				fmt.Printf("    Bytes [0:2] = % X → maybe version? %d.%02d\n",
					payload[:2], int(payload[1]), int(payload[0]))
			}
			if len(payload) >= 6 {
				fmt.Printf("    Bytes [0:6] = % X → maybe MAC? %s\n",
					payload[:6], formatMac(payload[:6]))
			}
			if len(payload) >= 8 {
				// Try as version info
				fmt.Printf("    Bytes [0:2] as fw version: %d.%02d\n",
					int(payload[1]), int(payload[0]))
				fmt.Printf("    Bytes [2:4] as CSR version: %d.%02d\n",
					int(payload[3]), int(payload[2]))
				fmt.Printf("    Bytes [4:6] as hotword version: %d.%02d\n",
					int(payload[5]), int(payload[4]))
			}
			// Print all bytes individually
			fmt.Println("  All payload bytes:")
			for i, b := range payload {
				fmt.Printf("    [%d] 0x%02X (%d)\n", i, b, b)
			}
		}
	}

	_ = binary.LittleEndian
}

func formatMac(b []byte) string {
	if len(b) < 6 {
		return ""
	}
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
		b[0], b[1], b[2], b[3], b[4], b[5])
}
