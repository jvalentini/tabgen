package main

import (
	"fmt"
	"os"

	"github.com/justin/tabgen/cmd"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	var err error
	switch os.Args[1] {
	case "scan":
		err = cmd.Scan()
	case "generate":
		tool := ""
		if len(os.Args) > 2 {
			tool = os.Args[2]
		}
		err = cmd.Generate(tool)
	case "list":
		showAll := len(os.Args) > 2 && os.Args[2] == "--all"
		err = cmd.List(showAll)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("tabgen - generate tab completions by analyzing CLI tools")
	fmt.Println()
	fmt.Println("Usage: tabgen <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  scan              Scan $PATH for executable tools")
	fmt.Println("  generate [tool]   Generate completions (all tools if none specified)")
	fmt.Println("  list [--all]      List discovered tools (parseable only by default)")
	fmt.Println("  help              Show this help message")
}
