package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/google/prog-edu-assistant/uploadserver"
	"github.com/google/prog-edu-assistant/queue"
)

var (
	port        = flag.Int("port", 8000, "The port to serve HTTP/S.")
	useHTTPS    = flag.Bool("use_https", false, "If true, use HTTPS instead of HTTP.")
	sslCertFile = flag.String("ssl_cert_file", "localhost.crt",
		"The path to the signed SSL server certificate.")
	sslKeyFile = flag.String("ssl_key_file", "localhost.key",
		"The path to the SSL server key.")
	uploadDir   = flag.String("upload_dir", "uploads", "The directory to write uploaded notebooks.")
	disableCORS = flag.Bool("disable_cors", false, "If true, disables CORS browser checks. "+
		"This is currently necessary to enable uploads from Jupyter notebooks, but unfortunately "+
		"it also makes the server vulnerable to XSRF attacks. Use with care.")
	queueSpec      = flag.String("queue_spec", "amqp://guest:guest@localhost:5672/",
		"The spec of the queue to connect to.")
	autograderQueue = flag.String("autograder_queue", "autograde",
		"The name of the autograder queue to send work requests.")
	reportQueue = flag.String("report_queue", "report",
		"The name of the queue to listen for the reports.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	q, err := queue.Open(*queueSpec)
	if err != nil {
		return fmt.Errorf("error opening queue %q: %s", *queueSpec, err)
	}
	ch, err := q.Receive(*reportQueue)
	if err != nil {
		return fmt.Errorf("error receiving on queue %q: %s", *autograderQueue, err)
	}
	s := uploadserver.New(uploadserver.Options{
		UploadDir:   *uploadDir,
		DisableCORS: *disableCORS,
		Channel: q,
		QueueName: *autograderQueue,
	})
	addr := ":" + strconv.Itoa(*port)
	protocol := "http"
	if *useHTTPS {
		protocol = "https"
	}
	fmt.Printf("\n  Serving on %s://localhost%s\n\n", protocol, addr)
	if *useHTTPS {
		return s.ListenAndServeTLS(addr, *sslCertFile, *sslKeyFile)
	}
	go s.ListenForReports(ch)
	return s.ListenAndServe(addr)
}
