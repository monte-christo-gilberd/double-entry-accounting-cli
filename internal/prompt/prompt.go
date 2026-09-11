package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var reader = bufio.NewReader(os.Stdin)

func ReadLine(text string) (string, error) {
	fmt.Print(text)

	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
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
		fmt.Println(" cannot be empty.")
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
		fmt.Println(" input must be a number.")
		return 0, false, nil
	}

	return convInput, true, nil
}

func ReadIntDefault(text string, def int) (int, error) {

	input, err := ReadLine(text)
	if err != nil {
		return 0, err
	}

	if input == "" {
		return def, nil
	}

	convInput, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println(" input not inputalid, use default:", def)
		return def, nil
	}

	return convInput, nil
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
		return 0, false, err
	}

	return convInput, true, nil
}

func ReadYesNo(text string) (bool, error) {

	input, err := ReadLine(text)
	if err != nil {
		return false, err
	}

	input = strings.ToLower(input)

	return input == "y" || input == "yes", nil

}
