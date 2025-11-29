package main

import (
	"fmt"
	"os"
)

// Ensures gofmt doesn't remove the "fmt" and "os" imports in stage 1 (feel free to remove this!)
var _ = fmt.Fprint
var _ = os.Stdout

func main() {
	var input string

	for {
		fmt.Fprint(os.Stdout, "$ ")
		fmt.Scanln(&input)
		if input == "exit" {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stdout, "%s: command not found\n", input)
	}

}
