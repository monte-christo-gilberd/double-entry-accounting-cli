package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ErrInputClosed is returned when stdin is closed (EOF), e.g. piped input
// runs out. Menus treat it as "go back / exit" instead of reprompting,
// which would otherwise spin forever on EOF.
var ErrInputClosed = errors.New("input closed")

var reader = bufio.NewReader(os.Stdin)

func ReadLine(text string) (string, error) {
	fmt.Print(text)

	line, err := reader.ReadString('\n')
	if err != nil {
		// bufio returns buffered data together with EOF when stdin closes
		// mid-line (e.g. piped input without trailing newline). Preserve
		// that data instead of discarding it as "input closed".
		if errors.Is(err, io.EOF) && strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line), nil
		}
		return "", mapReadError(err)
	}
	return strings.TrimSpace(line), nil
}

// mapReadError maps a closed stdin to ErrInputClosed (so errors.Is works)
// and wraps any other read failure with context.
func mapReadError(err error) error {
	if errors.Is(err, io.EOF) {
		return ErrInputClosed
	}
	return fmt.Errorf("read input: %w", err)
}

func ReadRequiredLine(text string) (string, error) {
	for {
		input, err := ReadLine(text)
		if err != nil {
			return "", err
		}
		if input != "" {
			return input, nil
		}
		fmt.Println("  Input cannot be empty.")
	}
}

func ReadLineDefault(text, def string) (string, error) {

	input, err := ReadLine(text)
	if err != nil {
		return "", err
	}

	if input == "" {
		return def, nil
	}

	return input, nil
}

func ReadInt(text string) (int, bool, error) {

	input, err := ReadLine(text)
	if err != nil {
		return 0, false, err
	}

	if input == "" {
		return 0, false, nil
	}

	convInput, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println("  input must be a number.")
		return 0, false, nil
	}

	return convInput, true, nil
}

func ReadIntDefault(text string, def int) (int, error) {
	for {
		input, err := ReadLine(text)
		if err != nil {
			return 0, err
		}

		value, ok := parseIntDefault(input, def)
		if ok {
			return value, nil
		}
		fmt.Println("  invalid input, enter a number or leave empty for default:", def)
	}
}

// parseIntDefault interprets one raw line: empty means the default, a valid
// number is accepted, anything else reports ok=false so the caller reprompts.
// Trims whitespace so direct calls behave like ReadIntDefault (which trims
// via ReadLine first).
func parseIntDefault(input string, def int) (int, bool) {
	if strings.TrimSpace(input) == "" {
		return def, true
	}
	convInput, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		return 0, false
	}
	return convInput, true
}

func ReadFloat(text string) (float64, bool, error) {

	input, err := ReadLine(text)
	if err != nil {
		return 0, false, err
	}

	if input == "" {
		return 0, false, nil
	}

	convInput, err := strconv.ParseFloat(input, 64)
	if err != nil {
		fmt.Println("  input must be a number.")
		return 0, false, nil
	}

	return convInput, true, nil
}

func ReadYesNo(text string) (bool, error) {
	for {
		input, err := ReadLine(text)
		if err != nil {
			return false, err
		}

		switch strings.ToLower(strings.TrimSpace(input)) {
		case "y", "yes":
			return true, nil
		case "n", "no", "":
			return false, nil
		default:
			fmt.Println("  Please answer y or n.")
		}
	}
}
