package main

import (
	"flag"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/go-yaml/yaml"
	"github.com/google/prog-edu-assistant/assign/notebook"
)

var (
	command = flag.String("command", "parse", "The command to execute.")
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
			fmt.Printf("  %s\t%s\n", name, cmd.Help)
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

var (
	assignmentMetadataRegex = regexp.MustCompile("(?m)^# ASSIGNMENT METADATA")
	exerciseMetadataRegex   = regexp.MustCompile("(?m)^# EXERCISE METADATA")
	tripleBacktickedRegex   = regexp.MustCompile("(?ms)^```.*^```")
	solutionBeginRegex      = regexp.MustCompile("(?m)^# BEGIN SOLUTION")
	solutionEndRegex        = regexp.MustCompile("(?m)^# END SOLUTION")
	promptBeginRegex        = regexp.MustCompile("(?m)^# BEGIN PROMPT")
	promptEndRegex          = regexp.MustCompile("(?m)^# END PROMPT")
	testRegex               = regexp.MustCompile("(?m)^# TEST")
	unittestBeginRegex      = regexp.MustCompile("(?m)^# BEGIN UNITTEST")
	unittestEndRegex        = regexp.MustCompile("(?m)^# END UNITTEST")
	autotestBeginRegex      = regexp.MustCompile("(?m)^# BEGIN AUTOTEST")
	autotestEndRegex        = regexp.MustCompile("(?m)^# END AUTOTEST")
)

func studentCommand() error {
	n, err := notebook.Parse(*input)
	assignmentMetadata := make(map[string]interface{})
	exerciseMetadata := make(map[string]interface{})
	n, err = n.MapCells(func(cell *notebook.Cell) (*notebook.Cell, error) {
		if cell.Type == "markdown" {
			var outputs []string
			mm := tripleBacktickedRegex.FindAllStringIndex(cell.Source, -1)
			replace := false
			for i, m := range mm {
				if len(outputs) == 0 {
					outputs = append(outputs, cell.Source[0:m[0]])
				}
				text := cell.Source[m[0]+3 : m[1]-3]
				if assignmentMetadataRegex.MatchString(text) {
					fmt.Printf("%q", text)
					err := yaml.Unmarshal([]byte(text), &assignmentMetadata)
					if err != nil {
						return nil, fmt.Errorf("error parsing ASSIGNMENT METADATA: %s", err)
					}
					replace = true
				}
				if exerciseMetadataRegex.MatchString(text) {
					exerciseMetadata = make(map[string]interface{})
					fmt.Printf("%q", text)
					err := yaml.Unmarshal([]byte(text), &exerciseMetadata)
					if err != nil {
						return nil, fmt.Errorf("error parsing EXERCISE METADATA: %s", err)
					}
					replace = true
				}
				if i < len(mm)-1 {
					outputs = append(outputs, cell.Source[m[1]:mm[i+1][0]])
				} else {
					outputs = append(outputs, cell.Source[m[1]:])
				}
			}
			if replace {
				cell.Source = strings.Join(outputs, "")
			}
		}
		if cell.Type != "code" {
			return cell, nil
		}
		if mbeg := solutionBeginRegex.FindAllStringIndex(cell.Source, -1); mbeg != nil {
			mend := solutionEndRegex.FindAllStringIndex(cell.Source, -1)
			if len(mbeg) != len(mend) {
				return nil, fmt.Errorf("cell has mismatched number of BEGIN SOLUTION and END SOLUTION, %d != %d", len(mbeg), len(mend))
			}
			var outputs []string
			for i, m := range mbeg {
				if i == 0 {
					outputs = append(outputs, cell.Source[0:m[0]])
				}
				// TODO(salikh): Fix indentation and add more heuristics.
				outputs = append(outputs, "...\n")
				if i < len(mbeg)-1 {
					outputs = append(outputs, cell.Source[m[1]:mbeg[i+1][0]])
				} else {
					outputs = append(outputs, cell.Source[m[1]:])
				}
			}
			return &notebook.Cell{
				Type:     "code",
				Metadata: exerciseMetadata,
				Source:   strings.Join(outputs, ""),
			}, nil
		}
		// Skip any test cells.
		if unittestBeginRegex.MatchString(cell.Source) ||
			autotestBeginRegex.MatchString(cell.Source) {
			return nil, nil
		}
		return cell, nil
	})
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
