package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" and "os" imports in stage 1 (feel free to remove this!)
var _ = fmt.Fprint
var _ = os.Stdout

func findExecutable(name string) (string, error) {
	exe, err := exec.LookPath(name)
	return exe, err

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
		tokens := strings.Split(inp, " ")
		command := tokens[0]
		if command == "exit" {
			os.Exit(0)
		} else if command == "echo" {
			for i := 1; i < len(tokens); i++ {
				fmt.Printf("%s ", tokens[i])
			}
			fmt.Println()
		} else if command == "type" {
			if slices.Contains(BuiltinCommands, tokens[1]) {
				fmt.Printf("%s is a shell builtin\n", tokens[1])
			} else if fPath, err := findExecutable(tokens[1]); err == nil {
				fmt.Printf("%s is %s\n", tokens[1], fPath)
			} else {
				fmt.Fprintf(os.Stdout, "%s: not found\n", tokens[1])
			}
		} else {
			fmt.Fprintf(os.Stdout, "%s: command not found\n", command)
		}
	}
}
