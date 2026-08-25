package spp

import (
	"bytes"
	"fmt"
	"io"
	"time"
)

// ProgressCallback is called during firmware flashing to report write progress.
// bytesWritten is the number of bytes sent so far, totalBytes is the
// total firmware size.
type ProgressCallback func(bytesWritten, totalBytes int)

// StepCallback is called at each step of the flashing process.
// step is 1-based, total is the total number of steps.
type StepCallback func(step, total int, name string)

// Flasher handles the firmware flashing process over SPP.
type Flasher struct {
	client       *Client
	timings      UpgradeTimings
	progress     ProgressCallback
	stepCallback StepCallback
}

// NewFlasher creates a new firmware flasher.
func NewFlasher(client *Client, timings UpgradeTimings) *Flasher {
	return &Flasher{
		client:  client,
		timings: timings,
	}
}

// SetProgressCallback sets a callback function for write progress reporting.
func (f *Flasher) SetProgressCallback(cb ProgressCallback) {
	f.progress = cb
}

// SetStepCallback sets a callback for step-level notifications.
func (f *Flasher) SetStepCallback(cb StepCallback) {
	f.stepCallback = cb
}

// Flash performs the complete firmware flashing sequence:
//  1. Connect (NOP handshake)
//  2. Erase SQIF flash
//  3. Write firmware data (in 1024-byte chunks)
//  4. Validate written data
//  5. Apply OTA update
//
// The reader should provide the raw .ota firmware binary data.
func (f *Flasher) Flash(firmwareData io.Reader, firmwareSize int) error {
	const totalSteps = 5

	// Step 1: Connect (NOP handshake)
	f.notifyStep(1, totalSteps, "连接握手")
	if err := f.client.Connect(); err != nil {
		return fmt.Errorf("step 1 connect: %w", err)
	}

	// Step 2: Erase SQIF
	f.notifyStep(2, totalSteps, "擦除闪存")
	if err := f.eraseSqif(); err != nil {
		return fmt.Errorf("step 2 erase: %w", err)
	}

	// Step 3: Write firmware data
	f.notifyStep(3, totalSteps, "写入数据")
	if err := f.writeSqif(firmwareData, firmwareSize); err != nil {
		return fmt.Errorf("step 3 write: %w", err)
	}

	// Step 4: Validate SQIF
	f.notifyStep(4, totalSteps, "校验数据")
	if err := f.validateSqif(); err != nil {
		return fmt.Errorf("step 4 validate: %w", err)
	}

	// Step 5: Apply OTA
	f.notifyStep(5, totalSteps, "应用固件")
	if err := f.applyOta(); err != nil {
		return fmt.Errorf("step 5 apply: %w", err)
	}

	// Close connection
	_ = f.client.Close()

	return nil
}

// notifyStep fires the step callback if set.
func (f *Flasher) notifyStep(step, total int, name string) {
	if f.stepCallback != nil {
		f.stepCallback(step, total, name)
	}
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

		if err := f.client.WriteCommand(CmdWriteSqif, buf[:n]); err != nil {
			return fmt.Errorf("write sqif at offset %d: %w", bytesWritten, err)
		}

		resp, err := f.client.ReadResponse()
		if err != nil {
			return fmt.Errorf("write ack at offset %d: %w", bytesWritten, err)
		}
		if !resp.StatusCode.IsOk() {
			return fmt.Errorf("write failed at offset %d: %s", bytesWritten, resp.StatusCode)
		}

		bytesWritten += n
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
func (f *Flasher) applyOta() error {
	f.client.SetTimeout(time.Duration(TimeoutApplyMs) * time.Millisecond)

	if err := f.client.WriteCommand(CmdApplyOta, []byte{0x00}); err != nil {
		return fmt.Errorf("apply ota write: %w", err)
	}

	time.Sleep(1 * time.Second)
	return nil
}

// EraseSqifExported is an exported wrapper of eraseSqif for GUI use.
func (f *Flasher) EraseSqifExported() error { return f.eraseSqif() }

// WriteSqifExported is an exported wrapper of writeSqif for GUI use.
func (f *Flasher) WriteSqifExported(data []byte, totalSize int) error {
	return f.writeSqif(bytes.NewReader(data), totalSize)
}

// ValidateSqifExported is an exported wrapper of validateSqif for GUI use.
func (f *Flasher) ValidateSqifExported() error { return f.validateSqif() }

// ApplyOtaExported is an exported wrapper of applyOta for GUI use.
func (f *Flasher) ApplyOtaExported() error { return f.applyOta() }