package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/justin/tabgen/cmd"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]
	args := os.Args[2:]

	var err error
	switch command {
	case "scan":
		err = cmd.Scan()

	case "generate":
		fs := flag.NewFlagSet("generate", flag.ExitOnError)
		force := fs.Bool("force", false, "force regeneration")
		fs.BoolVar(force, "f", false, "force regeneration (shorthand)")
		workers := fs.Int("workers", 0, "number of concurrent workers (default: NumCPU)")
		fs.IntVar(workers, "w", 0, "number of concurrent workers (shorthand)")
		fs.Usage = func() {
			fmt.Fprintln(os.Stderr, "Usage: tabgen generate [tool] [-f|--force] [-w|--workers N]")
			fs.PrintDefaults()
		}
		if err := fs.Parse(args); err != nil {
			os.Exit(1)
		}
		opts := cmd.GenerateOptions{Force: *force, Workers: *workers}
		if fs.NArg() > 0 {
			opts.Tool = fs.Arg(0)
		}
		err = cmd.Generate(opts)

	case "list":
		fs := flag.NewFlagSet("list", flag.ExitOnError)
		showAll := fs.Bool("all", false, "show all tools")
		fs.Usage = func() {
			fmt.Fprintln(os.Stderr, "Usage: tabgen list [--all]")
			fs.PrintDefaults()
		}
		if err := fs.Parse(args); err != nil {
			os.Exit(1)
		}
		err = cmd.List(*showAll)

	case "install":
		fs := flag.NewFlagSet("install", flag.ExitOnError)
		skipTimer := fs.Bool("skip-timer", false, "skip systemd timer setup")
		fs.Usage = func() {
			fmt.Fprintln(os.Stderr, "Usage: tabgen install [--skip-timer]")
			fs.PrintDefaults()
		}
		if err := fs.Parse(args); err != nil {
			os.Exit(1)
		}
		err = cmd.Install(*skipTimer)

	case "uninstall":
		fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
		keepData := fs.Bool("keep-data", false, "keep data directory")
		fs.Usage = func() {
			fmt.Fprintln(os.Stderr, "Usage: tabgen uninstall [--keep-data]")
			fs.PrintDefaults()
		}
		if err := fs.Parse(args); err != nil {
			os.Exit(1)
		}
		err = cmd.Uninstall(*keepData)

	case "status":
		err = cmd.Status()

	case "exclude":
		fs := flag.NewFlagSet("exclude", flag.ExitOnError)
		fs.Usage = func() {
			fmt.Fprintln(os.Stderr, "Usage: tabgen exclude <action> [pattern]")
			fmt.Fprintln(os.Stderr, "Actions: list, add, remove, clear")
		}
		if err := fs.Parse(args); err != nil {
			os.Exit(1)
		}
		action := ""
		pattern := ""
		if fs.NArg() > 0 {
			action = fs.Arg(0)
		}
		if fs.NArg() > 1 {
			pattern = fs.Arg(1)
		}
		err = cmd.Exclude(action, pattern)

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
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
	fmt.Println("  generate [tool] [-f] [-w N]  Generate completions (-f force, -w workers)")
	fmt.Println("  list [--all]            List discovered tools")
	fmt.Println("  install [--skip-timer]  Set up symlinks, timer, and shell hooks")
	fmt.Println("  uninstall [--keep-data] Remove TabGen installation")
	fmt.Println("  status                  Show installation status")
	fmt.Println("  exclude <action>        Manage exclusion list (list/add/remove/clear)")
	fmt.Println("  help                    Show this help message")
}
