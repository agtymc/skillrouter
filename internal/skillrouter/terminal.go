package skillrouter

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func stty(args ...string) (string, error) {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func withRawTerminal(run func() error) error {
	oldState, err := stty("-g")
	if err != nil {
		return fmt.Errorf("failed to read terminal state: %w", err)
	}
	if _, err := stty("raw", "-echo", "opost", "onlcr"); err != nil {
		return fmt.Errorf("failed to switch terminal to raw mode: %w", err)
	}
	defer func() {
		_, _ = stty(oldState)
		fmt.Print("\r\033[0m")
	}()

	return run()
}

func readByte(buf []byte) (int, error) {
	n, err := os.Stdin.Read(buf)
	if err != nil && err != io.EOF {
		return 0, err
	}
	return n, err
}
