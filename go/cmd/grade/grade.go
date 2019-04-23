// Binary grade runs the autograder manually without a daemon.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/google/prog-edu-assistant/autograder"
)

var (
	autograderDir = flag.String("autograder_dir", "tmp",
		"The root directory of autograder scripts.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ag := autograder.New(*autograderDir)
	for _, filename := range flag.Args() {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("error reading %q: %s", filename, err)
		}
		report, err := ag.Grade(b)
		if err != nil {
			return fmt.Errorf("error grading %q: %s", filename, err)
		}
		fmt.Println(string(report))
	}
	return nil
}
