package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
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

// FindExecutablesInPath returns the names (not full paths)
// of executables in PATH that start with the given prefix.
func FindExecutablesInPath() ([]string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return nil, nil
	}

	paths := filepath.SplitList(pathEnv)
	results := []string{}
	seen := make(map[string]bool)

	isWindows := runtime.GOOS == "windows"

	for _, dir := range paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			name := entry.Name()

			// determine executability
			if isWindows {
				// allowed Windows executable extensions
				ext := strings.ToLower(filepath.Ext(name))
				if ext != ".exe" && ext != ".bat" && ext != ".cmd" && ext != ".com" && ext != ".ps1" {
					continue
				}
			} else {
				// check Unix exec bit
				info, err := entry.Info()
				if err != nil {
					continue
				}
				mode := info.Mode()
				if mode&0111 == 0 { // executable for user/group/other
					continue
				}
			}

			// avoid duplicates caused by repeated PATH dirs
			if !seen[name] {
				seen[name] = true
				results = append(results, name)
			}
		}
	}

	return results, nil
}

func removeDuplicateStrings(slice []string) []string {
	encountered := map[string]bool{} // Map to store encountered strings
	result := []string{}             // Slice to store unique strings

	for _, str := range slice {
		if !encountered[str] { // If the string has not been encountered before
			encountered[str] = true      // Mark it as encountered
			result = append(result, str) // Add it to the result slice
		}
	}
	return result
}

// LongestCommonPrefixRunes returns the longest common prefix of a slice of
// rune slices ([]rune). If the input is empty, it returns nil.
func LongestCommonPrefixRunes(items [][]rune) []rune {
	if len(items) == 0 {
		return nil
	}

	prefix := items[0]

	for i := 1; i < len(items); i++ {
		prefix = lcpRunesTwo(prefix, items[i])
		if len(prefix) == 0 {
			return nil
		}
	}

	// Return a copy so modifying the result doesn’t modify the original input
	out := make([]rune, len(prefix))
	copy(out, prefix)
	return out
}

// lcpRunesTwo returns the LCP of two rune slices.
func lcpRunesTwo(a, b []rune) []rune {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	i := 0
	for i < minLen && a[i] == b[i] {
		i++
	}
	return a[:i]
}

// handleCatWithIO implements cat with custom I/O streams for use in pipelines.
func handleCatWithIO(files []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(files) == 0 {
		// Read from stdin
		_, err := io.Copy(stdout, stdin)
		return err
	}

	for _, path := range files {
		var r io.Reader
		if path == "-" {
			r = stdin
		} else {
			f, err := os.Open(path)
			if err != nil {
				if pathErr, ok := err.(*os.PathError); ok && os.IsNotExist(pathErr.Err) {
					fmt.Fprintf(stderr, "cat: %s: No such file or directory\n", pathErr.Path)
				}
				return err
			}
			defer f.Close()
			r = f
		}

		if _, err := io.Copy(stdout, r); err != nil {
			return err
		}
	}
	return nil
}

func handleHistoryCommand(args []string, stdout io.WriteCloser) error {
	var err error
	if len(args) > 1 {
		switch args[0] {
		case "-r":
			readHistoryFromFile(args[1])
		case "-w":
			fileToWrite, err := os.OpenFile(args[1], os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
			if err == nil {
				defer fileToWrite.Close()
				for _, historyCmd := range history {
					fileToWrite.WriteString(historyCmd + "\n")
				}
			}
		case "-a":
			fileToAppend, err := os.OpenFile(args[1], os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err == nil {
				defer fileToAppend.Close()
				for i := lastSavedHistory; i < len(history); i++ {
					fileToAppend.WriteString(history[i] + "\n")
				}
			}
			lastSavedHistory = len(history)
		}

	} else {
		var limit = len(history)
		if len(args) > 0 {
			limit, err = strconv.Atoi(args[0])
		}

		if err == nil {
			for i := len(history) - limit; i < len(history); i++ {
				fmt.Fprintf(stdout, "    %d %s\n", i+1, history[i])
			}
		}
	}

	return err
}

func readHistoryFromFile(filePath string) {
	content, err := os.ReadFile(filePath)
	prevHistory := strings.TrimRight(string(content), "\r\n")
	if err == nil {
		historyData := strings.Split(prevHistory, "\n")
		for _, historyCmd := range historyData {
			rl.SaveHistory(historyCmd)
			history = append(history, historyCmd)
		}
	}
}
