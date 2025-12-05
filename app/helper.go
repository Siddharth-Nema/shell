package main

import (
	"io"
	"os"
	"os/exec"
	"slices"
)

func findExecutable(name string) (string, error) {
	exe, err := exec.LookPath(name)
	return exe, err

}

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
