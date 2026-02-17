package main

import (
	"os"
	"github.com/braunmar/worktree/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
