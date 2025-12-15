package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/chzyer/readline"
)

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

func makeCompleter() *CommandsCompleter {
	var cmds []string
	cmds = append(cmds, BuiltinCommands...)
	pathCmds, err := FindExecutablesInPath()
	if err == nil {
		cmds = append(cmds, pathCmds...)
	}
	cmds = removeDuplicateStrings(cmds)
	completer := &CommandsCompleter{
		Commands:        cmds,
		CaseInsensitive: true,
		noOfTabs:        0,
		prevLine:        []rune(""),
	}
	return completer
}

func isStdinTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func handleCommand(command string, args []string) {
	var err error
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
}

func runPipeline(cmds [][]string) error {
	n := len(cmds)
	if n == 0 {
		return nil
	}

	procs := make([]*exec.Cmd, 0, n)
	var prevRd *os.File

	for i, argv := range cmds {
		if len(argv) == 0 {
			return fmt.Errorf("empty command at index %d", i)
		}

		cmd := exec.Command(argv[0], argv[1:]...)

		if prevRd != nil {
			cmd.Stdin = prevRd
		} else {
			cmd.Stdin = os.Stdin
		}

		if i < n-1 {
			r, w, err := os.Pipe()
			if err != nil {
				if prevRd != nil {
					prevRd.Close()
				}
				return fmt.Errorf("pipe creation failed: %w", err)
			}
			cmd.Stdout = w

			if prevRd != nil {
				prevRd.Close()
			}

			if err := cmd.Start(); err != nil {
				w.Close()
				r.Close()
				return fmt.Errorf("start failed for %v: %w", argv, err)
			}

			w.Close()
			prevRd = r
		} else {
			cmd.Stdout = os.Stdout

			if err := cmd.Start(); err != nil {
				if prevRd != nil {
					prevRd.Close()
				}
				return fmt.Errorf("start failed for %v: %w", argv, err)
			}

			if prevRd != nil {
				prevRd.Close()
				prevRd = nil
			}
		}

		cmd.Stderr = os.Stderr
		procs = append(procs, cmd)
	}

	var firstErr error
	for _, cmd := range procs {
		if err := cmd.Wait(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func main() {
	useReadline := false
	var rl *readline.Instance

	if isStdinTerminal() {
		config := &readline.Config{
			Prompt:          "$ ",
			AutoComplete:    makeCompleter(),
			InterruptPrompt: "^C",
			EOFPrompt:       "exit",
		}
		var err error
		rl, err = readline.NewEx(config)
		if err == nil {
			useReadline = true
			defer rl.Close()
		}
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		var inp string
		var err error
		if useReadline {
			line, errRead := rl.Readline()
			if errRead == readline.ErrInterrupt {
				continue
			}
			if errRead == io.EOF {
				return
			}
			if errRead != nil {
				return
			}
			inp = line
		} else {
			fmt.Fprint(os.Stdout, "$ ")
			inp, err = reader.ReadString('\n')
			if err != nil {
				return
			}
			if strings.Contains(inp, "\t") {
				before := strings.Split(inp, "\t")[0]
				comp := autoComplete(before)
				fmt.Printf("\r$ %s\n", comp)
				inp = comp
			}
		}

		inp = strings.TrimSpace(inp)
		inp = strings.ReplaceAll(inp, "\a", "")

		if inp == "" {
			continue
		}

		// Tokenize and handle redirections
		tokens := tokenize(inp)
		outputFile, errorFile, filteredTokens := getOutputFiles(tokens)

		// ✅ APPLY REDIRECTIONS
		if outputFile != nil {
			os.Stdout = outputFile
		}
		if errorFile != nil {
			os.Stderr = errorFile
		}

		// Split by pipe token
		var commands [][]string
		currentCmd := []string{}

		for _, token := range filteredTokens {
			if token == "|" {
				if len(currentCmd) > 0 {
					commands = append(commands, currentCmd)
					currentCmd = []string{}
				}
			} else {
				currentCmd = append(currentCmd, token)
			}
		}
		if len(currentCmd) > 0 {
			commands = append(commands, currentCmd)
		}

		if len(commands) == 0 {
			// Restore stdout/stderr
			if outputFile != nil {
				outputFile.Close()
				os.Stdout = STDOUT
			}
			if errorFile != nil {
				errorFile.Close()
				os.Stderr = STDERR
			}
			continue
		}

		// ✅ CHECK IF SINGLE COMMAND OR PIPELINE
		if len(commands) == 1 {
			// Single command - use handleCommand for builtins
			command := commands[0][0]
			args := []string{}
			if len(commands[0]) > 1 {
				args = commands[0][1:]
			}
			handleCommand(command, args)
		} else {
			// Pipeline - use runPipeline
			runPipeline(commands)
		}

		// ✅ RESTORE STDOUT/STDERR
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
