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
var STDERR = os.Stderr

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

func getOutputFiles(tokens []string) (*os.File, *os.File, []string) {
	var outputFilePath string = ""
	var errorFilePath string = ""
	filteredTokens := []string{}
	var outputMode int = os.O_TRUNC

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token == ">" || token == "1>" || token == ">>" || token == "1>>" {
			if i+1 < len(tokens) {
				outputFilePath = tokens[i+1]
				i++ // Skip the filename
				if token == ">>" || token == "1>>" {
					outputMode = os.O_APPEND
				}
			}
		} else if token == "2>" {
			if i+1 < len(tokens) {
				errorFilePath = tokens[i+1]
				i++ // Skip the filename
			}
		} else {
			filteredTokens = append(filteredTokens, token)
		}
	}

	var outputFile, errorFile *os.File

	if outputFilePath != "" {
		f, err := os.OpenFile(outputFilePath, os.O_WRONLY|os.O_CREATE|outputMode, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open %s: %v\n", outputFilePath, err)
		} else {
			outputFile = f
		}
	}
	if errorFilePath != "" {
		f, err := os.OpenFile(errorFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open %s: %v\n", errorFilePath, err)
		} else {
			errorFile = f
		}
	}

	return outputFile, errorFile, filteredTokens
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

		outputFile, errorFile, filteredTokens := getOutputFiles(tokens)
		command := filteredTokens[0]
		args := filteredTokens[1:]

		// Redirect stdout if needed
		if outputFile != nil {
			os.Stdout = outputFile
		}
		if errorFile != nil {
			os.Stderr = errorFile
		}

		switch command {
		case "exit":
			os.Exit(0)
		case "cat":
			err = handleCat(args)
			if err != nil {
				if pathErr, ok := err.(*os.PathError); ok && os.IsNotExist(pathErr.Err) {
					fmt.Fprintf(os.Stderr, "cat: %s: No such file or directory\n", pathErr.Path)
				} else {
					fmt.Fprintf(os.Stderr, "cat: %s\n", err)
				}
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
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin
				err := cmd.Run() // Use Run() instead of CombinedOutput()
				if err != nil {
					if errorFile == nil {
						fmt.Fprintf(os.Stderr, "%s: command failed: %v\n", command, err)
					}
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
		if errorFile != nil {
			errorFile.Close()
			os.Stderr = STDERR
		}
	}
}
