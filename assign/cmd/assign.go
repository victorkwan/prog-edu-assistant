package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/google/prog-edu-assistant/assign/notebook"
)

var (
	command = flag.String("command", "", "The command to execute.")
	input   = flag.String("input", "",
		"The file name of the input master notebook.")
	output = flag.String("output", "",
		"The file name of the output. If empty, output is written to stdout.")
)

type commandDesc struct {
	Help string
	Func func() error
}

var commands = map[string]commandDesc{
	"parse":   commandDesc{"Try parsing the input", parseCommand},
	"student": commandDesc{"Extract student notebook", studentCommand},
}

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if *command == "" {
		fmt.Printf("List of known commands:\n")
		for name, cmd := range commands {
			fmt.Printf("  %s   \t%s\n", name, cmd.Help)
		}
		return fmt.Errorf("command is not specified with --command.")
	}
	cmd, ok := commands[*command]
	if !ok {
		return fmt.Errorf("command %q is not defined", *command)
	}
	return cmd.Func()
}

func parseCommand() error {
	n, err := notebook.Parse(*input)
	if err != nil {
		return err
	}
	fmt.Printf("%d cells\n", len(n.Cells))
	for _, cell := range n.Cells {
		fmt.Printf("%s: %s\n", cell.Type, cell.Source)
		/*
			for name, val := range cell.Outputs {
				fmt.Printf("%s: %s\n", name, val)
			}
		*/
		fmt.Println("--")
	}
	fmt.Printf("nbformat %d minor %d\n", n.NBFormat, n.NBFormatMinor)
	return nil
}

func studentCommand() error {
	n, err := notebook.Parse(*input)
	if err != nil {
		return err
	}
	n, err = n.ToStudent()
	if err != nil {
		return err
	}
	fmt.Printf("%d cells\n", len(n.Cells))
	for _, cell := range n.Cells {
		fmt.Printf("%s: %s\n", cell.Type, cell.Source)
		/*
			for name, val := range cell.Outputs {
				fmt.Printf("%s: %s\n", name, val)
			}
		*/
		fmt.Println("--")
	}
	return nil
}
