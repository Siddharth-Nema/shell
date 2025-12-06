package main

import (
	"io"
	"os"
	"os/exec"
	"slices"
)

// findExecutable searches for an executable named 'name' in the directories named by the PATH environment variable.
// It returns the full path to the executable if found, otherwise an error.
func findExecutable(name string) (string, error) {
	exe, err := exec.LookPath(name)
	return exe, err
}

// handleType implements the 'type' command, which identifies whether a command is a builtin or an external executable.
// It returns a description string and any error encountered.
func handleType(args []string) (output string, err error) {
	if len(args) == 0 {
		return "type: missing argument", nil
	}
	target := args[0]
	if slices.Contains(BuiltinCommands, target) {
		return (target + " is a shell builtin"), nil
	} else if fPath, err := findExecutable(target); err == nil {
		return (target + " is " + fPath), nil
	} else {
		return (target + ": not found"), nil
	}
}

// handleCat implements the 'cat' command, which concatenates and prints file contents to stdout.
// If no files are specified or "-" is used, it reads from stdin.
func handleCat(files []string) error {
	if len(files) == 0 {
		files = []string{"-"}
	}

	for _, path := range files {
		var r io.Reader
		if path == "-" {
			r = os.Stdin
		} else {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			r = f
		}

		if _, err := io.Copy(os.Stdout, r); err != nil {
			return err
		}
	}
	return nil
}

// tokenize splits the input string into tokens, handling single quotes, double quotes, and escape sequences.
// It returns a slice of tokens with quotes and escape characters processed appropriately.
func tokenize(inp string) []string {
	var token string
	var tokens []string
	inSingleQuotes := false
	inDoubleQuotes := false
	inEscapeSequence := false

	for i := 0; i < len(inp); i++ {

		if inEscapeSequence {
			if inDoubleQuotes {
				if inp[i] != '"' && inp[i] != '\\' {
					token += "\\"
				}
			}
			token += string(inp[i])
			inEscapeSequence = false
			continue
		}

		if inp[i] == '\\' && !inSingleQuotes {
			inEscapeSequence = true
			continue
		}

		if inp[i] == '\'' && !inDoubleQuotes {
			inSingleQuotes = !inSingleQuotes
			continue
		}

		if inp[i] == '"' && !inSingleQuotes {
			inDoubleQuotes = !inDoubleQuotes
			continue
		}

		if inp[i] == ' ' {
			if inSingleQuotes || inDoubleQuotes {
				token += string(inp[i])
			} else {
				if len(token) > 0 {
					tokens = append(tokens, token)
					token = ""
				}
			}

		} else {
			token += string(inp[i])
		}
	}

	if len(token) > 0 {
		tokens = append(tokens, token)
		token = ""
	}

	return tokens
}
