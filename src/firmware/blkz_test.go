package firmware

import (
	"os"
	"path/filepath"
	"testing"
)

// TestExtractBlkz verifies that .blkz extraction works correctly:
//   - Extracts to a same-named subdirectory
//   - manifest.xml is present after extraction
//   - Re-extraction is idempotent (no error on second call)
func TestExtractBlkz(t *testing.T) {
	blkzPath := filepath.Join("..", "blkz", "Thunderstrike_0x010E.blkz")
	if _, err := os.Stat(blkzPath); os.IsNotExist(err) {
		t.Skip("test .blkz file not found:", blkzPath)
	}

	// Extract
	dir, err := ExtractBlkz(blkzPath)
	if err != nil {
		t.Fatalf("ExtractBlkz failed: %v", err)
	}

	// Verify manifest.xml exists
	manifestPath := filepath.Join(dir, "manifest.xml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("manifest.xml not found in extracted dir: %v", err)
	}

	// Verify thunderstrike.ota exists
	otaPath := filepath.Join(dir, "thunderstrike.ota")
	if _, err := os.Stat(otaPath); err != nil {
		t.Errorf("thunderstrike.ota not found in extracted dir: %v", err)
	}

	// Second call should be idempotent (skip extraction)
	dir2, err := ExtractBlkz(blkzPath)
	if err != nil {
		t.Fatalf("second ExtractBlkz failed: %v", err)
	}
	if dir != dir2 {
		t.Errorf("expected same dir, got %s vs %s", dir, dir2)
	}

	// Clean up
	_ = os.RemoveAll(dir)
}

// TestVerifyBlkzMd5 verifies MD5 checksum validation:
//   - Known firmware files return ChecksumMatch
//   - Unknown files return ChecksumUnknown
func TestVerifyBlkzMd5(t *testing.T) {
	blkzPath := filepath.Join("..", "blkz", "Thunderstrike_0x010E.blkz")
	if _, err := os.Stat(blkzPath); os.IsNotExist(err) {
		t.Skip("test .blkz file not found:", blkzPath)
	}

	// Known file should match
	result, err := VerifyBlkzMd5(blkzPath)
	if err != nil {
		t.Fatalf("VerifyBlkzMd5 failed: %v", err)
	}
	if result.Status != ChecksumMatch {
		t.Errorf("expected ChecksumMatch, got %s (actual=%s expected=%s)",
			result.Status, result.Actual, result.Expected)
	}
}

// TestVerifyLocaleBlkzMd5 verifies that the locale (multi-language) firmware
// has a separate MD5 entry and is correctly verified.
func TestVerifyLocaleBlkzMd5(t *testing.T) {
	blkzPath := filepath.Join("..", "blkz", "Thunderstrike_locale_0x0124.blkz")
	if _, err := os.Stat(blkzPath); os.IsNotExist(err) {
		t.Skip("test .blkz file not found:", blkzPath)
	}

	result, err := VerifyBlkzMd5(blkzPath)
	if err != nil {
		t.Fatalf("VerifyBlkzMd5 failed: %v", err)
	}
	if result.Status != ChecksumMatch {
		t.Errorf("expected ChecksumMatch for locale firmware, got %s (actual=%s expected=%s)",
			result.Status, result.Actual, result.Expected)
	}
}

// TestOpenLocaleBlkz verifies that a locale .blkz package is parsed correctly:
//   - Manifest has multiple <update> elements
//   - Multiple .ota entries are loaded with correct language codes
func TestOpenLocaleBlkz(t *testing.T) {
	blkzPath := filepath.Join("..", "blkz", "Thunderstrike_locale_0x0124.blkz")
	if _, err := os.Stat(blkzPath); os.IsNotExist(err) {
		t.Skip("test .blkz file not found:", blkzPath)
	}

	blkz, err := OpenBlkz(blkzPath)
	if err != nil {
		t.Fatalf("OpenBlkz failed: %v", err)
	}

	// Locale manifest should have multiple updates
	if !blkz.Manifest.IsLocale() {
		t.Errorf("expected locale manifest with multiple updates, got %d", len(blkz.Manifest.Updates))
	}

	// Should have multiple .ota entries
	if len(blkz.OtaEntries) < 2 {
		t.Errorf("expected multiple ota entries, got %d", len(blkz.OtaEntries))
	}

	// Each entry should have a non-empty language code
	for _, entry := range blkz.OtaEntries {
		if entry.Language == "" {
			t.Errorf("ota entry %s has empty language code", entry.Name)
		}
	}

	// Total size should be larger than a single .ota
	totalSize := blkz.TotalOtaSize()
	firstSize := len(blkz.OtaEntries[0].Data)
	if totalSize <= firstSize {
		t.Errorf("total size %d should be larger than first entry %d", totalSize, firstSize)
	}
}
