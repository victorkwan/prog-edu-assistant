package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Options struct {
	UploadDir string
}

type Server struct {
	Options
	*mux.Router
}

func New(opts Options) *Server {
	r := mux.NewRouter()
	s := &Server{
		Options: opts,
		Router:  r,
	}
	r.Handle("/upload", handleError(s.handleUpload())).Methods("POST")
	r.Handle("/", handleError(s.uploadForm())).Methods("GET")
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads",
		http.FileServer(http.Dir(s.Options.UploadDir))))
	r.HandleFunc("/upload", s.handleOptions()).Methods("OPTIONS")
	return s
}

func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.Router)
}

func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	return http.ListenAndServeTLS(addr, certFile, keyFile, s.Router)
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

func (s *Server) handleOptions() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Println("OPTIONS ", req.URL.Path)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
	}
}

const maxUploadSize = 1048576

func (s *Server) handleUpload() httpHandleFuncWithError {
	return func(w http.ResponseWriter, req *http.Request) error {
		fmt.Println("POST ", req.URL.Path)
		req.Body = http.MaxBytesReader(w, req.Body, maxUploadSize)
		err := req.ParseMultipartForm(maxUploadSize)
		if err != nil {
			return fmt.Errorf("error parsing upload form: %s", err)
		}
		f, _, err := req.FormFile("notebook")
		if err != nil {
			return fmt.Errorf("no notebook file in the form: %s\nRequest %s", err, req)
		}
		defer f.Close()
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("error reading upload: %s", err)
		}
		// TODO(salikh): Add user identifier to the file name.
		filename := filepath.Join(s.UploadDir, uuid.New().String()+".ipynb")
		err = ioutil.WriteFile(filename, b, 0600)
		if err != nil {
			return fmt.Errorf("error writing uploaded file: %s", err)
		}
		rfilename, err := s.scheduleCheck(filename)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		fmt.Fprintf(w, "/uploads/"+rfilename)
		return nil
	}
}

// scheduleCheck runs a checker for the notebook specified as a filename,
// and returns a filename of the report that was generated.
func (s *Server) scheduleCheck(filename string) (string, error) {
	cmd := exec.Command("python", "../../exercises/helloworld-check.py", "--input_file", filename)
	log.Println(cmd)
	// TODO(salikh): Constrain memory.
	// TODO(salikh): Kill after timeout.
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error runing the checker: %s", err)
	}
	rfilename := filename + ".txt"
	err = ioutil.WriteFile(rfilename, out, 0600)
	if err != nil {
		return "", fmt.Errorf("error writing the report %q: %s", rfilename, err)
	}
	return filepath.Base(rfilename), nil
}

func (s *Server) uploadForm() httpHandleFuncWithError {
	return func(w http.ResponseWriter, req *http.Request) error {
		fmt.Println("GET ", req.URL.Path)
		//return uploadTmpl.Execute(w, nil)
		_, err := w.Write([]byte(uploadHTML))
		return err
	}
}

//var uploadTmpl = template.Must(template.New("upload").Parse(uploadHTML))

const uploadHTML = `<!DOCTYPE html>
<title>Upload form</title>
<form method="POST" action="/upload" enctype="multipart/form-data">
	<input type="file" name="notebook">
	<input type="submit" value="Upload">
</form>`
