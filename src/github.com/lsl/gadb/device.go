package gadb

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Device represents a connected Android device
type Device struct {
	Serial, Product, Model, Device string
}

// String returns a formatted representation of the device
func (d *Device) String() string {
	if d.Model != "" {
		return fmt.Sprintf("%s (%s)", d.Serial, d.Model)
	}
	return d.Serial
}

// readDevices reads the list of connected devices using adb devices -l
// It filters out offline devices and returns a slice of Device structs
func readDevices() []Device {
	cmd := exec.Command("adb", "devices", "-l")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	reader := bufio.NewReader(stdout)
	var devices []Device
	re := regexp.MustCompile(`\s+`)
	for {
		line, e := reader.ReadString('\n')
		if e != nil || io.EOF == e {
			break
		}
		s := strings.Trim(line, "\n")
		if !strings.HasPrefix(s, "List of devices") && s != "" {
			ss := re.Split(s, -1)
			if len(ss) >= 2 {
				// Skip offline devices
				if ss[1] == "offline" {
					continue
				}
				// Parse device info: serial [usb:]product:vendor [model] [device]
				// Fields vary by device, so handle gracefully
				dev := Device{
					Serial: ss[0],
				}
				// Product is usually at index 2 (after serial and status)
				if len(ss) > 2 {
					// Parse usb:product:vendor format
					parts := strings.Split(ss[2], ":")
					if len(parts) >= 3 {
						dev.Product = parts[1] // product id
					} else {
						dev.Product = ss[2]
					}
				}
				// Model is at index 3 if available
				if len(ss) > 3 {
					dev.Model = ss[3]
				}
				// Device field is at index 4 if available
				if len(ss) > 4 {
					dev.Device = ss[4]
				}
				// Only add device if serial is not empty
				if dev.Serial != "" {
					devices = append(devices, dev)
				}
			}
		}
	}
	return devices
}

// selectDevices provides an interactive menu for selecting devices
// Returns the selected device(s) based on user input
func selectDevices(devs []Device) []Device {
	count := len(devs)
	fmt.Println("Connected devices:")
	fmt.Println("  [0] All devices")
	for i := 0; i < count; i++ {
		fmt.Printf("  [%d] %s\n", i+1, devs[i].String())
	}
	fmt.Println("  [q] Exit")

	input := bufio.NewScanner(os.Stdin)
	fmt.Printf("Select device [1]: ")
	input.Scan()
	line := input.Text()

	// Default to first device if empty
	if line == "" {
		if count > 0 {
			return []Device{devs[0]}
		}
		fmt.Println("No devices available")
		os.Exit(1)
	}

	switch line {
	case "0":
		return devs
	case "q", "Q":
		fmt.Println("Exiting...")
		os.Exit(0)
	default:
		c, err := strconv.Atoi(line)
		if err != nil || c < 0 || c > count {
			fmt.Printf("Invalid input: %s, please try again\n", line)
			return selectDevices(devs)
		}
		if c == 0 {
			return devs
		}
		return []Device{devs[c-1]}
	}
	// This should never be reached, but return nil to satisfy compiler
	return nil
}

// ListDevices prints all connected devices
func ListDevices() {
	devices := readDevices()
	if len(devices) == 0 {
		fmt.Println("No device found")
		return
	}
	fmt.Println("Connected devices:")
	for i, d := range devices {
		fmt.Printf("  [%d] %s\n", i+1, d.String())
	}
}
