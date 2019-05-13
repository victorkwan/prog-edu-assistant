package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/google/prog-edu-assistant/uploadserver"
	"github.com/google/prog-edu-assistant/queue"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	port        = flag.Int("port", 8000, "The port to serve HTTP/S.")
	useHTTPS    = flag.Bool("use_https", false, "If true, use HTTPS instead of HTTP.")
	sslCertFile = flag.String("ssl_cert_file", "localhost.crt",
		"The path to the signed SSL server certificate.")
	sslKeyFile = flag.String("ssl_key_file", "localhost.key",
		"The path to the SSL server key.")
	disableCORS = flag.Bool("disable_cors", false, "If true, disables CORS browser checks. "+
		"This is currently necessary to enable uploads from Jupyter notebooks, but unfortunately "+
		"it also makes the server vulnerable to XSRF attacks. Use with care.")
	useOpenID = flag.Bool("use_openid", true, "If true, enable OpenID Connect authentication.")
	openIDServer = flag.String("openIDServer", "",
		"The host name of the Open ID Connect server. It is used for constructing "+
		"/.well-known/openid-configuration URL. If empty, google Open ID Connect "+
		"is assumed.")
	uploadDir   = flag.String("upload_dir", "uploads", "The directory to write uploaded notebooks.")
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
	var oauthConfig *oauth2.Config
	if *openIDServer != "" {

	} else {
    oauthConfig = &oauth2.Config{
      RedirectURL:  opts.ServerURL + "/callback",
      ClientID:     opts.ClientID,
      ClientSecret: opts.ClientSecret,
      Scopes:       []string{"openid", "email"},
      Endpoint:     google.Endpoint,
    },
	}
	delay := 500*time.Millisecond
	retryUntil := time.Now().Add(60*time.Second)
	var q *queue.Channel
	var ch <-chan []byte
	for {
		var err error
		q, err = queue.Open(*queueSpec)
		if err != nil {
			if time.Now().After(retryUntil) {
				return fmt.Errorf("error opening queue %q: %s", *queueSpec, err)
			}
			glog.V(1).Infof("error opening queue %q: %s, retrying in %s", *queueSpec, err, delay)
			time.Sleep(delay)
			delay = delay*2
			continue
		}
		ch, err = q.Receive(*reportQueue)
		if err != nil {
			return fmt.Errorf("error receiving on queue %q: %s", *autograderQueue, err)
		}
		break
	}
	addr := ":" + strconv.Itoa(*port)
	protocol := "http"
	if *useHTTPS {
		protocol = "https"
	}
	serverURL := fmt.Sprintf("%s://localhost%s", protocol, addr)
	s := uploadserver.New(uploadserver.Options{
		DisableCORS: *disableCORS,
		ServerURL: serverURL,
		UploadDir:   *uploadDir,
		Channel: q,
		QueueName: *autograderQueue,
		UseOpenID: *useOpenID,
		// TODO(salikh): Take the list of users as CVS file on command line.
		AllowedUsers: map[string]bool{"salikh@gmail.com": true, "salikh@google.com": true},
		AuthKey:      os.Getenv("SESSION_AUTH_KEY"),
		EncryptKey:   os.Getenv("SESSION_ENCRYPTION_KEY"),
		ServerURL:    os.Getenv("SERVER_URL"),
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	})
	fmt.Printf("\n  Serving on %s\n\n", serverURL)
	if *useHTTPS {
		return s.ListenAndServeTLS(addr, *sslCertFile, *sslKeyFile)
	}
	go s.ListenForReports(ch)
	return s.ListenAndServe(addr)
}
