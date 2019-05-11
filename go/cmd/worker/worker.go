// Binary worker is the daemon that runs inside the autograder worker docker
// image, accepts requests on the message queue, runs autograder scripts
// under nsjail, creates reports and posts reports back to the message queue.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/prog-edu-assistant/queue"
	"github.com/google/prog-edu-assistant/autograder"
	"github.com/golang/glog"
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
	nsjailPath = flag.String("nsjail_path", "/usr/local/bin/nsjail",
		"The path to nsjail.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if !filepath.IsAbs(*autograderDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		*autograderDir = filepath.Join(cwd, *autograderDir)
	}
	*autograderDir = filepath.Clean(*autograderDir)
	ag := autograder.New(*autograderDir)
	ag.NSJailPath = *nsjailPath
	var err error
	q, err := queue.Open(*queueSpec)
	if err != nil {
		return fmt.Errorf("error opening queue %q: %s", *queueSpec, err)
	}
	ch, err := q.Receive(*autograderQueue)
	if err != nil {
		return fmt.Errorf("error receiving on queue %q: %s", *autograderQueue, err)
	}
	glog.Infof("Listening on the queue %q", *autograderQueue)
	// Enter the main work loop
	for b := range ch {
		glog.V(5).Infof("Received %d bytes: %s", len(b), string(b))
		report, err := ag.Grade(b)
		if err != nil {
			// TODO(salikh): Add remote logging and monitoring.
			log.Println(err)
		}
		glog.V(3).Infof("Grade result %d bytes: %s", len(report), string(report))
		err = q.Post(*reportQueue, report)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}
