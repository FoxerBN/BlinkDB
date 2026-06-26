package network

import (
	"errors"
	"strings"
)

type Command struct {
	Name string
	Args []string
}

// TODO validations:
// - unknown command should return error
// - PING must have 0 arguments
// - STATUS must have 0 arguments
// - QUIT must have 0 arguments
// - EXIT must have 0 arguments
// - GET must have exactly 1 argument
// - DELETE must have exactly 1 argument
// - SET must have exactly 2 arguments for now
// - key cannot be empty
// - key should not contain spaces
// - later: support SET value with spaces

func ParseCommand(input string) (*Command, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, errors.New("invalid command")
	}

	parts := strings.Fields(input)

	name := strings.ToUpper(parts[0])
	args := parts[1:]

	return &Command{
		Name: name,
		Args: args,
	}, nil
}
