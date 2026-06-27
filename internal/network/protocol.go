package network

import (
	"errors"
	"strings"
)

type Command struct {
	Name string
	Args []string
}

func ParseCommand(input string) (*Command, error) {
	// Trim whitespace and check for empty input
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, errors.New("empty command")
	}

	// Split and normalize input
	parts := strings.Fields(input)
	name := strings.ToUpper(parts[0])
	args := parts[1:]

	switch name {
	case "PING", "STATUS", "QUIT", "EXIT":
		if len(args) != 0 {
			return nil, errors.New("command expects no arguments")
		}
	case "GET", "DELETE":
		if len(args) != 1 {
			return nil, errors.New("command expects exactly one argument")
		}
		if !validKey(args[0]) {
			return nil, errors.New("invalid key")
		}
	case "SET":
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

func validKey(key string) bool {
	return key != ""
}
