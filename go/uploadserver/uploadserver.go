package uploadserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"reflect"

	"github.com/golang/glog"
	"github.com/google/prog-edu-assistant/queue"
	"github.com/google/uuid"
)

type Options struct {
	// UploadDir specifies the directory to write uploaded files to
	// and to serve on /uploads.
	UploadDir string
	// DisableCORS specifies whether the server should disable CORS
	// (Cross-origin request sharing) checks in browser by adding
	// Access-Control-Allow-Origin:* HTTP header.
	DisableCORS bool
	// QueueName is the name of the queue to post uploads.
	QueueName string
	// Channel is the interface to the message queue.
	*queue.Channel
}

type Server struct {
	opts Options
	mux  *http.ServeMux
}

func New(opts Options) *Server {
	mux := http.NewServeMux()
	s := &Server{
		opts: opts,
		mux:  mux,
	}
	mux.Handle("/", handleError(s.uploadForm))
	mux.Handle("/upload", handleError(s.handleUpload))
	mux.Handle("/uploads/", http.StripPrefix("/uploads",
		http.FileServer(http.Dir(s.opts.UploadDir))))
	return s
}

func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	return http.ListenAndServeTLS(addr, certFile, keyFile, s.mux)
}

func handleError(fn func(http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := fn(w, req)
		if err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

type httpHandleFuncWithError func(http.ResponseWriter, *http.Request) error

const maxUploadSize = 1048576

func (s *Server) handleUpload(w http.ResponseWriter, req *http.Request) error {
	if s.opts.DisableCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
	}
	if req.Method == "OPTIONS" {
		log.Println("OPTIONS ", req.URL.Path)
		return nil
	}
	if req.Method != "POST" {
		return fmt.Errorf("Unsupported method %s on %s", req.Method, req.URL.Path)
	}
	fmt.Println("POST ", req.URL.Path)
	req.Body = http.MaxBytesReader(w, req.Body, maxUploadSize)
	err := req.ParseMultipartForm(maxUploadSize)
	if err != nil {
		return fmt.Errorf("error parsing upload form: %s", err)
	}
	f, _, err := req.FormFile("notebook")
	if err != nil {
		return fmt.Errorf("no notebook file in the form: %s\nRequest %s", err, req.URL)
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("error reading upload: %s", err)
	}
	// TODO(salikh): Add user identifier to the file name.
	submissionID := uuid.New().String()
	filename := filepath.Join(s.opts.UploadDir, submissionID+".ipynb")
	err = ioutil.WriteFile(filename, b, 0700)
	glog.V(3).Infof("Uploaded %d bytes", len(b))
	if err != nil {
		return fmt.Errorf("error writing uploaded file: %s", err)
	}
	// Store submission ID inside the metadata.
	data := make(map[string]interface{})
	err = json.Unmarshal(b, &data)
	if err != nil {
		return fmt.Errorf("could not parse submission as JSON: %s", err)
	}
	var metadata map[string]interface{}
	v, ok := data["metadata"]
	if ok {
		metadata, ok = v.(map[string]interface{})
	}
	if !ok {
		metadata = make(map[string]interface{})
		data["metadata"] = metadata
	}
	metadata["submission_id"] = submissionID
	b, err = json.Marshal(data)
	if err != nil {
		return err
	}
	glog.V(3).Infof("Checking %d bytes", len(b))
	err = s.scheduleCheck(b)
	if err != nil {
		return err
	}
	reportfilename := submissionID + ".txt"
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	fmt.Fprintf(w, "/uploads/"+ reportfilename)
	return nil
}

func (s *Server) scheduleCheck(content []byte) error {
	return s.opts.Channel.Post(s.opts.QueueName, content)
}

func (s *Server) ListenForReports(ch <-chan []byte) {
	for b := range ch {
		glog.V(3).Infof("Received %d byte report", len(b))
		data := make(map[string]interface{})
		err := json.Unmarshal(b, &data)
		if err != nil {
			glog.Errorf("data: %q, error: %s", string(b), err)
			continue
		}
		v, ok := data["submission_id"]
		if !ok {
			glog.Errorf("Report did not have submission_id: %#v", data)
			continue
		}
		submissionID, ok := v.(string)
		if !ok {
			glog.Errorf("submission_id was not a string, but %s",
				reflect.TypeOf(v))
			continue
		}
		// TODO(salikh): Write a pretty report instead.
		filename := filepath.Join(s.opts.UploadDir, submissionID + ".txt")
		err = ioutil.WriteFile(filename, b, 0775)
		if err != nil {
			glog.Errorf("Error writing to %q: %s", filename, err)
			continue
		}
	}
}

func (s *Server) uploadForm(w http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		return fmt.Errorf("Unsupported method %s on %s", req.Method, req.URL.Path)
	}
	fmt.Println("GET ", req.URL.Path)
	_, err := w.Write([]byte(uploadHTML))
	return err
}

const uploadHTML = `<!DOCTYPE html>
<title>Upload form</title>
<form method="POST" action="/upload" enctype="multipart/form-data">
	<input type="file" name="notebook">
	<input type="submit" value="Upload">
</form>`
