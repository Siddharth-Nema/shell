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
		args := tokens[1:]

		switch command {
		case "exit":
			os.Exit(0)

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
				fmt.Fprintf(os.Stdout, "%s: command not found\n", command)
			}
		}
	}
}
