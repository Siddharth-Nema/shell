package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/eiannone/keyboard"
)

// Ensures gofmt doesn't remove the "fmt" and "os" imports in stage 1 (feel free to remove this!)
var _ = fmt.Fprint
var _ = os.Stdout

var STDOUT = os.Stdout
var STDERR = os.Stderr

// getOutputFiles processes redirection operators (>, >>, 2>, 2>>) from tokens and opens the target files.
// It returns the output file, error file, and a filtered token slice without redirection operators.
func getOutputFiles(tokens []string) (*os.File, *os.File, []string) {
	var outputFilePath string = ""
	var errorFilePath string = ""
	filteredTokens := []string{}
	var outputMode int = os.O_TRUNC
	var errorMode int = os.O_TRUNC

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token == ">" || token == "1>" || token == ">>" || token == "1>>" {
			if i+1 < len(tokens) {
				outputFilePath = tokens[i+1]
				i++
				if token == ">>" || token == "1>>" {
					outputMode = os.O_APPEND
				}
			}
		} else if token == "2>" || token == "2>>" {
			if i+1 < len(tokens) {
				errorFilePath = tokens[i+1]
				i++
				if token == "2>>" {
					errorMode = os.O_APPEND
				}
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
		f, err := os.OpenFile(errorFilePath, os.O_WRONLY|os.O_CREATE|errorMode, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open %s: %v\n", errorFilePath, err)
		} else {
			errorFile = f
		}
	}

	return outputFile, errorFile, filteredTokens
}

func readInputWithCompletion() (string, error) {
	var input string

	for {
		char, key, err := keyboard.GetSingleKey()
		if err != nil {
			return "", err
		}

		switch key {
		case keyboard.KeyEnter:
			fmt.Println()
			return input, nil

		case keyboard.KeyTab:
			// 	// Auto-completion logic
			completed := autoComplete(input)
			if completed != input {
				// Clear current line and print completed
				fmt.Print("\r$ " + completed)
				input = completed
			}

		case keyboard.KeyBackspace, keyboard.KeyBackspace2:
			if len(input) > 0 {
				input = input[:len(input)-1]
				fmt.Print("\b \b") // Erase character
			}

		case keyboard.KeyCtrlC:
			return "", fmt.Errorf("interrupted")

		case keyboard.KeySpace:
			// Handle space explicitly
			input += " "
			fmt.Print(" ")

		default:
			if char != 0 && char >= 32 && char <= 126 {
				// Only printable ASCII characters
				input += string(char)
				fmt.Print(string(char))
			}
		}
	}
}

func autoComplete(partial string) string {
	tokens := tokenize(partial)
	lastToken := tokens[0]

	if len(tokens) == 0 {
		return partial
	}
	matches := []string{}

	if len(tokens) == 1 {
		for _, cmd := range BuiltinCommands {
			if strings.HasPrefix(cmd, lastToken) {
				matches = append(matches, cmd)
			}
		}
	}

	if len(matches) == 1 {
		tokens[len(tokens)-1] = matches[0]
		return strings.Join(tokens, " ") + " "
	}

	return partial

}

func main() {
	err := keyboard.Open()
	useKeyboard := err == nil
	if useKeyboard {
		defer keyboard.Close()
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Fprint(os.Stdout, "$ ")
		var inp string
		if useKeyboard {
			inp, err = readInputWithCompletion()
			if err != nil {
				break
			}
		} else {
			// Fallback to simple reading for non-interactive environments
			inp, err = reader.ReadString('\n')
			if err != nil {
				break
			}
		}
		inp = strings.TrimSpace(inp)
		tokens := tokenize(inp)
		if len(tokens) == 0 {
			continue
		}

		outputFile, errorFile, filteredTokens := getOutputFiles(tokens)
		command := filteredTokens[0]
		args := filteredTokens[1:]

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
				cmd.Run()
			} else {
				fmt.Fprintf(os.Stderr, "%s: command not found\n", command)
			}
		}

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
