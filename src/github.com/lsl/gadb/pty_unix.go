//go:build !windows

package gadb

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

// ExecWithPTY executes an adb command with PTY support for full interactivity
// This is needed for commands like 'adb shell' that require a terminal
func ExecWithPTY(deviceSerial string, args []string) error {
	// Build adb command with device serial
	adbArgs := append([]string{"-s", deviceSerial}, args...)
	cmd := exec.Command("adb", adbArgs...)

	// Start PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start PTY: %w", err)
	}
	// Make sure to close the pty at the end
	defer func() { _ = ptmx.Close() }()

	// Handle terminal size changes
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, unix.SIGWINCH)
	go func() {
		for range ch {
			_ = pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	defer signal.Stop(ch)

	// Initial terminal size
	_ = pty.InheritSize(os.Stdin, ptmx)

	// Set stdin to raw mode
	oldState, err := makeRaw(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to set terminal: %w", err)
	}
	defer restoreTerminal(os.Stdin, oldState)

	// Copy stdin to PTY and PTY to stdout
	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()

	_, err = io.Copy(os.Stdout, ptmx)
	if err != nil {
		return fmt.Errorf("error copying output: %w", err)
	}

	// Get command exit status
	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

// makeRaw puts the terminal into raw mode
func makeRaw(f *os.File) (*unix.Termios, error) {
	fd := int(f.Fd())
	oldState, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}

	newState := *oldState
	newState.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP |
		unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	newState.Oflag &^= unix.OPOST
	newState.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	newState.Cflag &^= unix.CSIZE | unix.PARENB
	newState.Cflag |= unix.CS8
	newState.Cc[unix.VMIN] = 1
	newState.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &newState); err != nil {
		return nil, err
	}

	return oldState, nil
}

// restoreTerminal restores the terminal to its previous state
func restoreTerminal(f *os.File, state *unix.Termios) error {
	if state == nil {
		return nil
	}
	fd := int(f.Fd())
	return unix.IoctlSetTermios(fd, unix.TCSETS, state)
}

// setupCommand configures a command for proper execution on Unix
// No special handling needed on Unix
func setupCommand(cmd *exec.Cmd) {
	// Unix handles signals properly by default
}
