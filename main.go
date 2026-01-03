package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("tabgen - generate tab completions from bash history")
		fmt.Println()
		fmt.Println("Usage: tabgen <command>")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  scan     Scan bash history for CLI tools")
		fmt.Println("  generate Generate completions for a tool")
		fmt.Println("  list     List discovered tools")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "scan":
		fmt.Println("TODO: scan bash history")
	case "generate":
		fmt.Println("TODO: generate completions")
	case "list":
		fmt.Println("TODO: list discovered tools")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
