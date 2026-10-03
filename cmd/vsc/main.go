package main

import (
	"github.com/HarveyZgit/alfred.vsc/internal/workflow"
	"os"
)

func main() { os.Exit(workflow.Run(os.Args[1:], os.Stdout, os.Stderr)) }
