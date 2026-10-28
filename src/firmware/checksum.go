// Package firmware — MD5 checksum verification for .blkz firmware files.
//
// To prevent flashing wrong/corrupted firmware (which would brick the
// controller), we maintain a built-in table of known-good MD5 hashes.
// Each entry maps the .blkz file name to its expected MD5.
//
// Using file name (not version hex) as the key allows distinguishing
// between standard and locale (multi-language) firmware that share
// the same version number but have different contents.
//
// When a new firmware version is released, add its MD5 here after verifying
// it comes from a trusted source.
package firmware

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// knownMd5 maps .blkz file name (e.g. "Thunderstrike_0x0112.blkz") to the
// expected MD5 of the file. These are verified against official NVIDIA firmware.
var knownMd5 = map[string]string{
	"thunderstrike_0x010e.blkz":         "4d3f6820f3d2ca8dd1bd9eb0cc08d5d3",
	"thunderstrike_0x0112.blkz":         "eedf8c187bf1b14602a1a7dd07fa310b",
	"thunderstrike_0x0121.blkz":         "8b7e0740d74e535bc229601ea67f1aa5",
	"thunderstrike_0x0124.blkz":         "d4acee6b52034c7043a8bbefc3620384",
	"thunderstrike_locale_0x0121.blkz":  "33684e7c51c3bb00046eefe7dcbe0232",
	"thunderstrike_locale_0x0124.blkz":  "a26dd47beae27cc259ade27666e8e872",
}

// ChecksumResult holds the result of an MD5 verification.
type ChecksumResult struct {
	Expected string // expected MD5 hex (lowercase), "" if unknown
	Actual   string // actual MD5 hex (lowercase)
	Status   ChecksumStatus
}

// ChecksumStatus describes the verification outcome.
type ChecksumStatus int

const (
	ChecksumUnknown   ChecksumStatus = iota // file not in known list
	ChecksumMatch                          // MD5 matches known-good value
	ChecksumMismatch                       // MD5 does NOT match — danger!
)

// String returns a human-readable status string.
func (s ChecksumStatus) String() string {
	switch s {
	case ChecksumMatch:
		return "校验通过"
	case ChecksumMismatch:
		return "校验失败"
	default:
		return "未知固件"
	}
}

// VerifyBlkzMd5 computes the MD5 of a .blkz file and compares it against
// the built-in known-good hash for that file name.
//
// The file name (base name) is used as the lookup key, so standard and
// locale firmware with the same version number are distinguished correctly.
func VerifyBlkzMd5(path string) (*ChecksumResult, error) {
	actual, err := computeFileMd5(path)
	if err != nil {
		return nil, fmt.Errorf("compute md5: %w", err)
	}

	filename := strings.ToLower(filepath.Base(path))
	expected, known := knownMd5[filename]

	result := &ChecksumResult{Actual: actual}
	if !known {
		result.Expected = ""
		result.Status = ChecksumUnknown
	} else {
		result.Expected = expected
		if strings.EqualFold(actual, expected) {
			result.Status = ChecksumMatch
		} else {
			result.Status = ChecksumMismatch
		}
	}

	return result, nil
}

// computeFileMd5 reads a file and returns its MD5 hex digest (lowercase).
func computeFileMd5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
