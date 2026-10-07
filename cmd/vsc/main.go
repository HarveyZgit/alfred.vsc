package main

import (
	"os"

	"github.com/HarveyZgit/alfred.vsc/internal/workflow"
)

func main() {
	os.Exit(workflow.Run(os.Args[1:], os.Stdout, os.Stderr))
}
