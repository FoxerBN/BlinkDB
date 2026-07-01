package main

import (
	"strings"
	"testing"

	"blinkdb/internal/network"
)

//* TestParseCommandValidatesKeys checks key validation and argument rules.
func TestParseCommandValidatesKeys(t *testing.T) {
	longKey := strings.Repeat("a", 513)

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "simple key", input: "GET user:123", wantErr: false},
		{name: "empty command", input: "   ", wantErr: true},
		{name: "missing key", input: "GET", wantErr: true},
		{name: "too long key", input: "GET " + longKey, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := network.ParseCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("network.ParseCommand(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

//* TestParseCommandAcceptsHelp checks that HELP parses with no arguments.
func TestParseCommandAcceptsHelp(t *testing.T) {
	command, err := network.ParseCommand("HELP")
	if err != nil {
		t.Fatalf("network.ParseCommand(HELP) error = %v", err)
	}
	if command.Name != "HELP" {
		t.Fatalf("command.Name = %q, want HELP", command.Name)
	}
}

//* TestParseCommandAcceptsExists checks that EXISTS parses its single key argument.
func TestParseCommandAcceptsExists(t *testing.T) {
	command, err := network.ParseCommand("EXISTS token")
	if err != nil {
		t.Fatalf("network.ParseCommand(EXISTS token) error = %v", err)
	}
	if command.Name != "EXISTS" {
		t.Fatalf("command.Name = %q, want EXISTS", command.Name)
	}
	if len(command.Args) != 1 || command.Args[0] != "token" {
		t.Fatalf("command.Args = %v, want [token]", command.Args)
	}
}
