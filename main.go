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
		opts := cmd.GenerateOptions{}
		for _, arg := range os.Args[2:] {
			if arg == "--force" || arg == "-f" {
				opts.Force = true
			} else if len(arg) > 0 && arg[0] != '-' {
				opts.Tool = arg
			}
		}
		err = cmd.Generate(opts)
	case "list":
		showAll := len(os.Args) > 2 && os.Args[2] == "--all"
		err = cmd.List(showAll)
	case "install":
		skipTimer := len(os.Args) > 2 && os.Args[2] == "--skip-timer"
		err = cmd.Install(skipTimer)
	case "uninstall":
		keepData := len(os.Args) > 2 && os.Args[2] == "--keep-data"
		err = cmd.Uninstall(keepData)
	case "status":
		err = cmd.Status()
	case "exclude":
		action := ""
		pattern := ""
		if len(os.Args) > 2 {
			action = os.Args[2]
		}
		if len(os.Args) > 3 {
			pattern = os.Args[3]
		}
		err = cmd.Exclude(action, pattern)
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
	fmt.Println("  scan                    Scan $PATH for executable tools")
	fmt.Println("  generate [tool] [-f]    Generate completions (-f to force regeneration)")
	fmt.Println("  list [--all]            List discovered tools")
	fmt.Println("  install [--skip-timer]  Set up symlinks, timer, and shell hooks")
	fmt.Println("  uninstall [--keep-data] Remove TabGen installation")
	fmt.Println("  status                  Show installation status")
	fmt.Println("  exclude <action>        Manage exclusion list (list/add/remove/clear)")
	fmt.Println("  help                    Show this help message")
}
