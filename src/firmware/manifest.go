package firmware

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// Manifest represents the manifest.xml structure inside a .blkz package.
//
// A standard manifest has a single <update> with one firmware file:
//   <blakemanifest>
//     <update accessory="THUNDERSTRIKE" fingerprint="...">
//       <thunderstrike name="thunderstrike.ota" version="010E" />
//     </update>
//   </blakemanifest>
//
// A locale (multi-language) manifest has multiple <update> elements,
// each with a different language and .ota file:
//   <blakemanifest>
//     <update accessory="THUNDERSTRIKE" fingerprint="...">
//       <thunderstrike name="thunderstrike_de_de.ota" version="0124" language="de_de" />
//     </update>
//     <update accessory="THUNDERSTRIKE" fingerprint="...">
//       <thunderstrike name="thunderstrike_fr_fr.ota" version="0124" language="fr_fr" />
//     </update>
//     ...
//   </blakemanifest>
type Manifest struct {
	XMLName xml.Name       `xml:"blakemanifest"`
	Updates []UpdateElement `xml:"update"`
}

// UpdateElement holds the firmware update metadata.
type UpdateElement struct {
	Accessory   string          `xml:"accessory,attr"`
	Fingerprint string          `xml:"fingerprint,attr"`
	Flags       string          `xml:"flags,attr"`
	Firmware    FirmwareElement `xml:"thunderstrike"`
}

// FirmwareElement holds the firmware file information.
type FirmwareElement struct {
	Name     string `xml:"name,attr"`
	Version  string `xml:"version,attr"`
	Language string `xml:"language,attr"` // only for locale firmware
}

// IsLocale returns true if this is a multi-language firmware entry.
func (f *FirmwareElement) IsLocale() bool {
	return f.Language != ""
}

// firstUpdate returns the first update element, or a zero value if none.
func (m *Manifest) firstUpdate() *UpdateElement {
	if len(m.Updates) > 0 {
		return &m.Updates[0]
	}
	return nil
}

// VersionHex returns the version as a hex string (e.g. "010E").
func (m *Manifest) VersionHex() string {
	if u := m.firstUpdate(); u != nil {
		return u.Firmware.Version
	}
	return ""
}

// VersionMajor returns the major version number.
// For version "010E": major = 0x01 = 1.
func (m *Manifest) VersionMajor() int {
	v := m.VersionHex()
	if len(v) < 2 {
		return 0
	}
	var n int
	fmt.Sscanf(v[:2], "%02X", &n)
	return n
}

// VersionMinor returns the minor version number.
// For version "010E": minor = 0x0E = 14.
func (m *Manifest) VersionMinor() int {
	v := m.VersionHex()
	if len(v) < 4 {
		return 0
	}
	var n int
	fmt.Sscanf(v[2:4], "%02X", &n)
	return n
}

// VersionString returns a human-readable version string (e.g. "1.14").
func (m *Manifest) VersionString() string {
	return fmt.Sprintf("%d.%d", m.VersionMajor(), m.VersionMinor())
}

// AccessoryType returns the accessory type (e.g. "THUNDERSTRIKE").
func (m *Manifest) AccessoryType() string {
	if u := m.firstUpdate(); u != nil {
		return strings.ToUpper(u.Accessory)
	}
	return ""
}

// IsLocale returns true if the manifest contains multi-language firmware.
func (m *Manifest) IsLocale() bool {
	return len(m.Updates) > 1
}

// Languages returns the list of language codes from a locale manifest.
// Returns [""] for standard (non-locale) firmware.
func (m *Manifest) Languages() []string {
	var langs []string
	for _, u := range m.Updates {
		langs = append(langs, u.Firmware.Language)
	}
	return langs
}

// FirmwareNames returns the list of .ota file names from the manifest.
func (m *Manifest) FirmwareNames() []string {
	var names []string
	for _, u := range m.Updates {
		names = append(names, u.Firmware.Name)
	}
	return names
}

// SupportsDowngrade returns true if the manifest contains downgrade flags.
func (m *Manifest) SupportsDowngrade() bool {
	if u := m.firstUpdate(); u != nil {
		if u.Flags == "" {
			return false
		}
		return strings.Contains(u.Flags, "downgrade") ||
			strings.Contains(u.Flags, "parcel=true")
	}
	return false
}

// ParseManifest parses manifest.xml content.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := xml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest xml: %w", err)
	}
	return &m, nil
}
