package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"go.bug.st/serial"
)

// scanCmd scans for Bluetooth serial port devices.
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for Bluetooth SPP devices",
	Long: `Scan for Bluetooth devices that expose the SPP (Serial Port Profile)
service. This is needed to find the controller's Bluetooth MAC address
for firmware flashing.

On Windows, this lists available COM ports. On Linux, this lists
paired Bluetooth devices.

Make sure the controller is paired to your computer before scanning.`,
	RunE: runScan,
}

// scanCmd is registered in root.go init().
// Kept for legacy/scripted use: `thunderstrike-controller-tool.exe scan`
func runScan(cmd *cobra.Command, args []string) error {
	fmt.Println("Scanning for serial port devices...")

	ports, err := serial.GetPortsList()
	if err != nil {
		return fmt.Errorf("list serial ports: %w", err)
	}

	if len(ports) == 0 {
		fmt.Println("No serial ports found.")
		fmt.Println()
		fmt.Println("Tips:")
		fmt.Println("  - Pair the controller with your computer first")
		fmt.Println("  - On Windows, check 'Bluetooth and other devices' settings")
		fmt.Println("  - On Linux, use 'bluetoothctl' to pair")
		return nil
	}

	// Filter unique port names
	seen := make(map[string]bool)
	var uniquePorts []string
	for _, p := range ports {
		if !seen[p] {
			seen[p] = true
			uniquePorts = append(uniquePorts, p)
		}
	}

	fmt.Printf("Found %d serial port(s):\n\n", len(uniquePorts))
	for _, port := range uniquePorts {
		// Try to get port details
		desc := "(unknown)"
		fmt.Printf("  %-20s  %s\n", port, desc)
	}

	fmt.Println()
	fmt.Println("To flash firmware, use the COM port or device path with:")
	fmt.Println("  nvtsct flash --port <PORT> -f <firmware.blkz>")
	return nil
}

// formatScanResult formats a scan result for display.
func formatScanResult(name, mac string, rssi int, lastSeen time.Time) string {
	return fmt.Sprintf("  %-30s  %s  RSSI=%d  %s",
		name, mac, rssi, lastSeen.Format("15:04:05"))
}
