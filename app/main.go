package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" and "os" imports in stage 1 (feel free to remove this!)
var _ = fmt.Fprint
var _ = os.Stdout

var STDOUT = os.Stdout

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

func getOutputFile(tokens []string) (*os.File, []string) {
	var filePath string = ""
	var redirectIndex = -1

	for index, token := range tokens {
		if token == ">" || token == "1>" {
			if index+1 < len(tokens) {
				filePath = tokens[index+1]
				redirectIndex = index
				break
			}
		}
	}

	// Filter out redirection tokens from args
	filteredTokens := tokens
	if redirectIndex >= 0 {
		filteredTokens = append(tokens[:redirectIndex], tokens[redirectIndex+2:]...)
	}

	if filePath != "" {
		f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open %s: %v\n", filePath, err)
			return nil, filteredTokens
		}
		return f, filteredTokens
	}

	return nil, filteredTokens
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Fprint(os.Stdout, "$ ")
		inp, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		inp = strings.TrimSpace(inp)
		tokens := tokenize(inp)
		if len(tokens) == 0 {
			continue
		}

		outputFile, filteredTokens := getOutputFile(tokens)
		command := filteredTokens[0]
		args := filteredTokens[1:]

		// Redirect stdout if needed
		if outputFile != nil {
			os.Stdout = outputFile
		}

		switch command {
		case "exit":
			os.Exit(0)
		case "cat":
			err = handleCat(args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s \n", err)
			}
		case "echo":
			for i := 0; i < len(args); i++ {
				fmt.Printf("%s ", args[i])
			}
			fmt.Println()

		case "type":
			output, err := handleType(args)
			if err == nil {
				fmt.Println(output)
			}
		case "pwd":
			pwd, err := os.Getwd()
			if err == nil {
				fmt.Println(pwd)
			}
		case "cd":
			if len(args) > 0 {
				var newDir string
				if args[0] == "~" {
					newDir, err = os.UserHomeDir()
					if err != nil {
						log.Fatal(err)
					}
				} else {
					newDir = args[0]
				}
				err := os.Chdir(newDir)
				if err != nil {
					fmt.Printf("cd: %s: No such file or directory\n", newDir)
				}
			}
		default:
			if _, err := findExecutable(command); err == nil {
				cmd := exec.Command(command, args...)
				output, err := cmd.CombinedOutput()
				if err == nil {
					fmt.Printf("%s", output)
				}
			} else {
				fmt.Fprintf(os.Stderr, "%s: command not found\n", command)
			}
		}

		// Restore stdout after command execution
		if outputFile != nil {
			outputFile.Close()
			os.Stdout = STDOUT
		}
	}
}
