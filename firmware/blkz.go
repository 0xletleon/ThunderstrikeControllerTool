// Package firmware handles parsing of .blkz firmware package files
// used by NVIDIA Thunderstrike controller.
//
// .blkz files are ZIP archives (no compression) containing:
//   - manifest.xml      — firmware manifest with version info
//   - manifest.xml.sig  — RSA signature of the manifest
//   - thunderstrike.ota — the firmware binary
//   - thunderstrike.ota.sig — RSA signature of the firmware
package firmware

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// OtaEntry represents a single .ota firmware binary and its signature.
type OtaEntry struct {
	Name     string // file name, e.g. "thunderstrike.ota" or "thunderstrike_de_de.ota"
	Language string // language code, e.g. "de_de", or "" for standard firmware
	Data     []byte // raw .ota binary content
	Sig      []byte // RSA signature (not sent to device)
}

// BlkzFile represents an opened .blkz firmware package.
//
// Standard firmware has one OtaEntries[0] with Language="".
// Locale firmware has multiple OtaEntries, one per language.
type BlkzFile struct {
	Path        string
	Manifest    *Manifest
	ManifestRaw []byte
	ManifestSig []byte
	OtaEntries  []OtaEntry
}

// OtaData returns the first .ota binary (for standard firmware, this is
// the only one; for locale firmware, it's the first language).
func (b *BlkzFile) OtaData() []byte {
	if len(b.OtaEntries) > 0 {
		return b.OtaEntries[0].Data
	}
	return nil
}

// OtaFileSize returns the size of the first .ota binary.
func (b *BlkzFile) OtaFileSize() int {
	return len(b.OtaData())
}

// TotalOtaSize returns the combined size of all .ota binaries.
func (b *BlkzFile) TotalOtaSize() int {
	total := 0
	for _, e := range b.OtaEntries {
		total += len(e.Data)
	}
	return total
}

// OtaVersion returns the firmware version string from the manifest.
func (b *BlkzFile) OtaVersion() string {
	if b.Manifest == nil {
		return ""
	}
	return b.Manifest.VersionString()
}

// OpenBlkz opens a .blkz firmware package and extracts its contents.
func OpenBlkz(path string) (*BlkzFile, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open blkz: %w", err)
	}
	defer r.Close()

	result := &BlkzFile{Path: path}

	// Build a lookup of firmware file names from the manifest so we can
	// match .ota files to their language codes.
	fwNames := make(map[string]string) // file name -> language

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s in blkz: %w", f.Name, err)
		}

		switch {
		case f.Name == "manifest.xml":
			data, err := io.ReadAll(rc)
			if err != nil {
				rc.Close()
				return nil, fmt.Errorf("read manifest: %w", err)
			}
			result.ManifestRaw = data
			result.Manifest, err = ParseManifest(data)
			if err != nil {
				rc.Close()
				return nil, fmt.Errorf("parse manifest: %w", err)
			}
			// Build name -> language map from manifest
			for _, u := range result.Manifest.Updates {
				fwNames[u.Firmware.Name] = u.Firmware.Language
			}
		case f.Name == "manifest.xml.sig":
			data, err := io.ReadAll(rc)
			if err != nil {
				rc.Close()
				return nil, fmt.Errorf("read manifest sig: %w", err)
			}
			result.ManifestSig = data
		case strings.HasSuffix(f.Name, ".ota"):
			data, err := io.ReadAll(rc)
			if err != nil {
				rc.Close()
				return nil, fmt.Errorf("read %s: %w", f.Name, err)
			}
			result.OtaEntries = append(result.OtaEntries, OtaEntry{
				Name:     f.Name,
				Language: fwNames[f.Name], // "" for standard firmware
				Data:     data,
			})
		case strings.HasSuffix(f.Name, ".ota.sig"):
			// Match .sig to its .ota entry by name
			otaName := strings.TrimSuffix(f.Name, ".sig")
			data, err := io.ReadAll(rc)
			if err != nil {
				rc.Close()
				return nil, fmt.Errorf("read %s: %w", f.Name, err)
			}
			for i := range result.OtaEntries {
				if result.OtaEntries[i].Name == otaName {
					result.OtaEntries[i].Sig = data
					break
				}
			}
		}
		rc.Close()
	}

	if len(result.OtaEntries) == 0 {
		return nil, fmt.Errorf("blkz missing .ota firmware file(s)")
	}
	if result.Manifest == nil {
		return nil, fmt.Errorf("blkz missing manifest.xml")
	}

	return result, nil
}

// SaveOtaToFile writes the first .ota firmware binary to a temporary file
// and returns the path. The caller is responsible for deleting the file.
func (b *BlkzFile) SaveOtaToFile(dir string) (string, error) {
	return b.SaveEntryToFile(0, dir)
}

// SaveEntryToFile writes a specific .ota entry to a temporary file
// and returns the path. index selects which language entry (0 = first).
func (b *BlkzFile) SaveEntryToFile(index int, dir string) (string, error) {
	if index < 0 || index >= len(b.OtaEntries) {
		return "", fmt.Errorf("ota entry index %d out of range (have %d)", index, len(b.OtaEntries))
	}

	entry := b.OtaEntries[index]
	pattern := strings.TrimSuffix(entry.Name, ".ota") + "-*.ota"
	tmpFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(entry.Data); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("write temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// ExtractBlkz extracts a .blkz (ZIP) archive to a subdirectory named after
// the .blkz file (without extension). For example:
//
//	blkz/Thunderstrike_0x0112.blkz → blkz/Thunderstrike_0x0112/
//
// If the target directory already exists and contains all expected files,
// extraction is skipped (idempotent). Returns the target directory path.
func ExtractBlkz(blkzPath string) (string, error) {
	// Target dir = blkzPath without .blkz extension
	ext := strings.ToLower(filepath.Ext(blkzPath))
	dir := strings.TrimSuffix(blkzPath, filepath.Ext(blkzPath))
	_ = ext // ext is always ".blkz"

	// Check if already extracted (manifest.xml exists)
	manifestPath := filepath.Join(dir, "manifest.xml")
	if _, err := os.Stat(manifestPath); err == nil {
		// Already extracted
		return dir, nil
	}

	// Create target directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create dir %s: %w", dir, err)
	}

	return extractBlkzToDir(blkzPath, dir)
}

// extractBlkzToDir extracts all files from a .blkz ZIP archive into dir.
func extractBlkzToDir(blkzPath, dir string) (string, error) {
	r, err := zip.OpenReader(blkzPath)
	if err != nil {
		return "", fmt.Errorf("open blkz: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// Security: prevent path traversal
		name := filepath.Base(f.Name)
		if name == "." || name == "" {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("open %s in blkz: %w", f.Name, err)
		}

		outPath := filepath.Join(dir, name)
		outFile, err := os.Create(outPath)
		if err != nil {
			rc.Close()
			return "", fmt.Errorf("create %s: %w", outPath, err)
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			outFile.Close()
			rc.Close()
			return "", fmt.Errorf("write %s: %w", outPath, err)
		}

		outFile.Close()
		rc.Close()
	}

	return dir, nil
}
