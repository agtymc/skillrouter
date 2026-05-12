package skillrouter

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func selectFromMenu(title string, items []string) (int, string, error) {
	if len(items) == 0 {
		return -1, "", errors.New("empty menu")
	}

	fmt.Print("\033[?1049h\033[?25l")
	oldState, err := stty("-g")
	if err != nil {
		fmt.Print("\033[?25h\033[?1049l")
		return -1, "", fmt.Errorf("failed to read terminal state: %w", err)
	}
	if _, err := stty("raw", "-echo"); err != nil {
		fmt.Print("\033[?25h\033[?1049l")
		return -1, "", fmt.Errorf("failed to switch terminal to raw mode: %w", err)
	}
	defer func() {
		_, _ = stty(oldState)
		fmt.Print("\033[H\033[2J\033[0m\033[?25h\033[?1049l")
	}()

	index := 0
	renderMenu(title, items, index)
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return -1, "esc", nil
			}
			return -1, "", err
		}
		if n == 0 {
			continue
		}

		b := buf[:n]
		switch {
		case len(b) == 1 && b[0] == 3:
			return -1, "esc", nil
		case len(b) == 1 && b[0] == 4:
			return -1, "esc", nil
		case len(b) == 1 && b[0] == 13:
			fmt.Print("\033[H\033[2J")
			return index, "enter", nil
		case len(b) == 1 && b[0] == 27:
			fmt.Print("\033[H\033[2J")
			return -1, "esc", nil
		case len(b) >= 3 && b[0] == 27 && b[1] == 91 && b[2] == 65:
			if index > 0 {
				index--
			}
			renderMenu(title, items, index)
		case len(b) >= 3 && b[0] == 27 && b[1] == 91 && b[2] == 66:
			if index < len(items)-1 {
				index++
			}
			renderMenu(title, items, index)
		}
	}
}

func renderMenu(title string, items []string, index int) {
	fmt.Print("\033[H\033[2J")
	fmt.Printf("%s\r\n\r\n", title)
	for i, item := range items {
		if i == index {
			fmt.Printf("\033[1;36m> %s\033[0m\r\n", item)
		} else {
			fmt.Printf("  %s\r\n", item)
		}
	}
	fmt.Print("\r\nUse Up/Down, Enter to select, Esc to go back\r\n")
}
