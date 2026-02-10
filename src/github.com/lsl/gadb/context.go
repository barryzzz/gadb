package gadb

import (
	"fmt"
)

// Context holds the state for a REPL session
type Context struct {
	// Available devices from the last device scan
	AvailableDevices []Device
	// Currently selected device for command execution
	CurrentDevice *Device
	// Command history
	History []string
	// Flag to indicate if the REPL should continue running
	Running bool
	// Exit code to return when exiting
	ExitCode int
}

// NewContext creates a new REPL context with the given devices
func NewContext(devices []Device) *Context {
	var current *Device
	if len(devices) == 1 {
		current = &devices[0]
	}
	return &Context{
		AvailableDevices: devices,
		CurrentDevice:    current,
		History:          make([]string, 0, 100),
		Running:          true,
		ExitCode:         0,
	}
}

// SetCurrentDevice sets the current device by index
func (c *Context) SetCurrentDevice(index int) error {
	if index < 0 || index >= len(c.AvailableDevices) {
		return fmt.Errorf("invalid device index: %d", index)
	}
	c.CurrentDevice = &c.AvailableDevices[index]
	return nil
}

// GetPrompt returns the current prompt string
func (c *Context) GetPrompt() string {
	if c.CurrentDevice != nil {
		return fmt.Sprintf("[GADB] %s > ", c.CurrentDevice.Serial)
	}
	return "[GADB] > "
}

// AddToHistory adds a command to the history
func (c *Context) AddToHistory(cmd string) {
	if cmd == "" {
		return
	}
	// Avoid duplicate consecutive entries
	if len(c.History) > 0 && c.History[len(c.History)-1] == cmd {
		return
	}
	c.History = append(c.History, cmd)
}

// RefreshDevices rescans for available devices
func (c *Context) RefreshDevices() {
	c.AvailableDevices = readDevices()
	if len(c.AvailableDevices) == 0 {
		fmt.Println("Warning: No devices found")
		c.CurrentDevice = nil
		return
	}
	// Try to maintain the current device selection
	if c.CurrentDevice != nil {
		for _, d := range c.AvailableDevices {
			if d.Serial == c.CurrentDevice.Serial {
				c.CurrentDevice = &d
				return
			}
		}
		// Current device disconnected, select first available
		c.CurrentDevice = &c.AvailableDevices[0]
	}
}

// EnsureDevice checks if a device is selected, exits if not
func (c *Context) EnsureDevice() bool {
	if c.CurrentDevice == nil {
		fmt.Println("No device selected. Use 'devices' and 'select' commands first.")
		return false
	}
	return true
}

// Stop marks the REPL for exit
func (c *Context) Stop(code int) {
	c.Running = false
	c.ExitCode = code
}
