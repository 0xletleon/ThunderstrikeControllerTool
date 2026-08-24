package spp

import (
	"fmt"
	"io"
	"time"
)

// ProgressCallback is called during firmware flashing to report progress.
// bytesWritten is the number of bytes sent so far, totalBytes is the
// total firmware size.
type ProgressCallback func(bytesWritten, totalBytes int)

// Flasher handles the firmware flashing process over SPP.
type Flasher struct {
	client   *Client
	timings  UpgradeTimings
	progress ProgressCallback
}

// NewFlasher creates a new firmware flasher.
func NewFlasher(client *Client, timings UpgradeTimings) *Flasher {
	return &Flasher{
		client:  client,
		timings: timings,
	}
}

// SetProgressCallback sets a callback function for progress reporting.
func (f *Flasher) SetProgressCallback(cb ProgressCallback) {
	f.progress = cb
}

// Flash performs the complete firmware flashing sequence:
//  1. Connect (NOP handshake)
//  2. Erase SQIF flash
//  3. Write firmware data (in 1024-byte chunks)
//  4. Validate written data
//  5. Apply OTA update
//  6. Wait for device restart
//
// The reader should provide the raw .ota firmware binary data.
func (f *Flasher) Flash(firmwareData io.Reader, firmwareSize int) error {
	// Step 1: Connect (NOP handshake)
	if err := f.client.Connect(); err != nil {
		return fmt.Errorf("step 1 connect: %w", err)
	}

	// Step 2: Erase SQIF
	if err := f.eraseSqif(); err != nil {
		return fmt.Errorf("step 2 erase: %w", err)
	}

	// Step 3: Write firmware data
	if err := f.writeSqif(firmwareData, firmwareSize); err != nil {
		return fmt.Errorf("step 3 write: %w", err)
	}

	// Step 4: Validate SQIF
	if err := f.validateSqif(); err != nil {
		return fmt.Errorf("step 4 validate: %w", err)
	}

	// Step 5: Apply OTA
	if err := f.applyOta(); err != nil {
		return fmt.Errorf("step 5 apply: %w", err)
	}

	// Step 6: Close connection
	// Original app: close → waitForDisconnect(30s) → reconnectDelay →
	// waitForConnect(120s) → waitForDeviceReady(30s)
	// On PC we can't auto-reconnect Bluetooth, so we just close and return.
	_ = f.client.Close()

	return nil
}

// eraseSqif sends the ERASE_SQIF command with parameter [0x00].
func (f *Flasher) eraseSqif() error {
	f.client.SetTimeout(time.Duration(TimeoutEraseMs) * time.Millisecond)

	resp, err := f.client.ExecuteCommand(CmdEraseSqif, []byte{0x00})
	if err != nil {
		return fmt.Errorf("erase sqif: %w", err)
	}
	if !resp.StatusCode.IsOk() {
		return fmt.Errorf("erase sqif failed: %s", resp.StatusCode)
	}
	return nil
}

// writeSqif sends the WRITE_SQIF command in chunks of MaxChunkSize bytes.
// Each chunk is sent and an ACK response is expected.
func (f *Flasher) writeSqif(data io.Reader, totalSize int) error {
	f.client.SetTimeout(time.Duration(TimeoutWriteMs) * time.Millisecond)

	buf := make([]byte, MaxChunkSize)
	bytesWritten := 0

	for {
		n, err := data.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read firmware data: %w", err)
		}
		if n == 0 {
			break
		}

		// Write chunk
		if err := f.client.WriteCommand(CmdWriteSqif, buf[:n]); err != nil {
			return fmt.Errorf("write sqif at offset %d: %w", bytesWritten, err)
		}

		// Wait for ACK
		resp, err := f.client.ReadResponse()
		if err != nil {
			return fmt.Errorf("write ack at offset %d: %w", bytesWritten, err)
		}
		if !resp.StatusCode.IsOk() {
			return fmt.Errorf("write failed at offset %d: %s",
				bytesWritten, resp.StatusCode)
		}

		bytesWritten += n

		// Report progress
		if f.progress != nil {
			f.progress(bytesWritten, totalSize)
		}
	}

	return nil
}

// validateSqif sends the VALIDATE_SQIF command.
func (f *Flasher) validateSqif() error {
	f.client.SetTimeout(time.Duration(TimeoutValidateMs) * time.Millisecond)

	resp, err := f.client.ExecuteCommand(CmdValidateSqif, nil)
	if err != nil {
		return fmt.Errorf("validate sqif: %w", err)
	}
	if !resp.StatusCode.IsOk() {
		return fmt.Errorf("validation failed: %s", resp.StatusCode)
	}
	return nil
}

// applyOta sends the APPLY_OTA command with parameter [0x00].
// After this command, the controller will restart with the new firmware.
func (f *Flasher) applyOta() error {
	f.client.SetTimeout(time.Duration(TimeoutApplyMs) * time.Millisecond)

	// Write command without waiting for response
	// (device will restart after receiving this)
	if err := f.client.WriteCommand(CmdApplyOta, []byte{0x00}); err != nil {
		return fmt.Errorf("apply ota write: %w", err)
	}

	// Wait 1 second for device to process
	time.Sleep(1 * time.Second)

	return nil
}
