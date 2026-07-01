package network

import (
	"errors"
	"strings"
	"unicode"
)

const maxKeyBytes = 512

type Command struct {
	Name string
	Args []string
}

//* ParseCommand takes a raw input string and returns a Command struct if the input is valid.
func ParseCommand(input string) (*Command, error) {
	//* Trim whitespace and check for empty input.
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, errors.New("empty command")
	}

	//* Split the input into parts and validate the command name.
	parts := strings.Fields(input)
	name := strings.ToUpper(parts[0])
	args := parts[1:]

	switch name {
	case "PING", "STATUS", "QUIT", "EXIT", "HELP", "CLEAR":
		// Connection/control commands are intentionally argument-free.
		if len(args) != 0 {
			return nil, errors.New("command expects no arguments")
		}
	case "GET", "DELETE", "EXISTS":
		//* Single-key commands need exactly one key.
		if len(args) != 1 {
			return nil, errors.New("command expects exactly one argument")
		}
		if !validKey(args[0]) {
			return nil, errors.New("invalid key")
		}
	case "SET":
		//* SET currently accepts exactly one key and one value token.
		if len(args) != 2 {
			return nil, errors.New("command expects exactly two arguments")
		}
		if !validKey(args[0]) {
			return nil, errors.New("invalid key")
		}

	default:
		return nil, errors.New("unknown command")
	}

	return &Command{
		Name: name,
		Args: args,
	}, nil
}

//* validKey keeps keys usable in the line-based protocol and cheap to store.
func validKey(key string) bool {
	if key == "" || len(key) > maxKeyBytes {
		return false
	}

	for _, r := range key {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}

	return true
}
