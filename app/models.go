package main

import (
	"io"
)

type pipeCmd struct {
	command string
	args    []string
	stdin   io.Reader
	stdout  io.WriteCloser
	stderr  io.Writer
}
