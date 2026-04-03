package main

import (
	"fmt"
	"os"

	"github.com/twinbird/mkdmy/internal/cli"
	"github.com/twinbird/mkdmy/internal/generator"
)

func main() {
	opts, helpRequested, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n\n", err)
		cli.PrintUsage(os.Stderr)
		os.Exit(2)
	}

	if helpRequested {
		cli.PrintUsage(os.Stdout)
		return
	}

	if err := generator.Generate(opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
