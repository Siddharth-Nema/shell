package main

import (
	"bufio"
	"fmt"
	"os"
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
		if command == "exit" {
			os.Exit(0)
		} else if command == "echo" {
			for i := 1; i < len(tokens); i++ {
				fmt.Printf("%s ", tokens[i])
			}
			fmt.Println()
		} else {
			fmt.Fprintf(os.Stdout, "%s: command not found\n", command)
		}

	}

}
