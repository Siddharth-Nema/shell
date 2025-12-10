package main

import (
	"fmt"
	"strings"
)

// CommandsCompleter completes from a provided list of commands/words.
type CommandsCompleter struct {
	Commands        []string
	CaseInsensitive bool
	noOfTabs        int
}

// Do implements readline.AutoCompleter.
// - line: full buffer as runes
// - pos:  cursor position in runes
// Returns:
// - suggestions as [][]rune (each suggestion must be the suffix to insert)
// - length: number of runes already shared (the prefix length)
func (c *CommandsCompleter) Do(line []rune, pos int) ([][]rune, int) {
	c.noOfTabs++
	// find start of current token (space-separated)
	start := pos
	for start > 0 {
		r := line[start-1]
		if r == ' ' || r == '\t' {
			break
		}
		start--
	}

	// prefix is what user has typed for this token
	prefixRunes := line[start:pos]
	prefix := string(prefixRunes)
	prefix = strings.ReplaceAll(prefix, "\a", "")

	if c.CaseInsensitive {
		prefix = strings.ToLower(prefix)
	}

	var out [][]rune
	for _, cmd := range c.Commands {
		match := cmd
		if c.CaseInsensitive {
			match = strings.ToLower(cmd)
		}
		if strings.HasPrefix(match, prefix) {
			// return the *suffix* (what to append), not the full word
			// convert prefix length to rune count correctly
			// prefixLen is the number of runes already present
			prefixLen := len([]rune(prefix))
			cmdRunes := []rune(cmd + " ")
			// append the remainder of cmd after the prefix
			out = append(out, cmdRunes[prefixLen:])
		}
	}
	if len(out) > 1 {

		if c.noOfTabs < 2 {
			out = nil
			out = append(out, []rune("\a"))
			return out, len(prefixRunes)
		} else {
			c.noOfTabs = 0
			fmt.Println()
			for _, suggestion := range out {
				fmt.Printf("%s ", prefix+string(suggestion))
			}
			fmt.Println()
			fmt.Printf("$ %s", prefix)
			out = nil
			return out, len(prefixRunes)
		}
	} else if len(out) == 1 {
		// length should be how many runes are already shared (prefix length)
		return out, len(prefixRunes)
	} else {
		out = append(out, []rune("\a"))
		return out, len(prefixRunes)
	}

}
