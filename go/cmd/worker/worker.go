// Binary worker is the daemon that runs inside the autograder worker docker
// image, accepts requests on the message queue, runs autograder scripts
// under nsjail, creates reports and posts reports back to the message queue.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/google/prog-edu-assistant/queue"
	"github.com/google/prog-edu-assistant/autograder"
)

var (
	queueSpec      = flag.String("queue_spec", "amqp://guest:guest@localhost:5672/",
		"The spec of the queue to connect to.")
	autograderQueue = flag.String("autograder_queue", "autograde",
		"The name of the autograder queue to listen to the work requests.")
	reportQueue = flag.String("report_queue", "report",
		"The name of the queue to post the reports.")
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
	var err error
	ag := autograder.New(*autograderDir)
	q, err := queue.Open(*queueSpec)
	if err != nil {
		return fmt.Errorf("error opening queue %q: %s", *queueSpec, err)
	}
	ch, err := q.Receive(*autograderQueue)
	if err != nil {
		return fmt.Errorf("error receiving on queue %q: %s", *autograderQueue, err)
	}
	// Enter the main work loop
	for b := range ch {
		report, err := ag.Grade(b)
		if err != nil {
			// TODO(salikh): Add remote logging and monitoring.
			log.Println(err)
		}
		err = q.Post(*reportQueue, report)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}
