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
	// 1. Najprv odstranis medzery/zalamovanie na zaciatku a konci vstupu.
	input = strings.TrimSpace(input)

	// 2. Prazdny vstup nie je platny command.
	if input == "" {
		return nil, errors.New("invalid command")
	}

	// 3. Fields rozdeli vstup podla medzier.
	// Priklad: "SET name Richard" -> []string{"SET", "name", "Richard"}
	parts := strings.Fields(input)

	// 4. Na velke pismena men iba nazov commandu.
	// Argumenty nemen, lebo su to data pouzivatela.
	name := strings.ToUpper(parts[0])
	args := parts[1:]

	// TODO: tu dopln validaciu podla name.
	// - unknown command ma vratit error
	// - PING, STATUS, QUIT, EXIT musia mat 0 argumentov
	// - GET a DELETE musia mat presne 1 argument
	// - SET musi mat presne 2 argumenty
	// - pri GET/DELETE/SET skontroluj key cez helper alebo jednoduche if-y
	// - key je typicky args[0], preto najprv over pocet argumentov
	//
	// Odporucany tvar:
	// switch name {
	// case "PING", "STATUS", "QUIT", "EXIT":
	//     // skontroluj len pocet argumentov
	// case "GET", "DELETE":
	//     // skontroluj pocet argumentov a key
	// case "SET":
	//     // skontroluj pocet argumentov a key
	// default:
	//     // vrat unknown command error
	// }

	return &Command{
		Name: name,
		Args: args,
	}, nil
}
