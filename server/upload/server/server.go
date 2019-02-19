package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
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
	r.Handle("/uploads/", http.StripPrefix("/uploads",
		http.FileServer(http.Dir(s.Options.UploadDir))))
	return s
}

func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.Router)
}

func handleError(fn func(http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := fn(w, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

type httpHandleFuncWithError func(http.ResponseWriter, *http.Request) error

const maxUploadSize = 1048576

func (s *Server) handleUpload() httpHandleFuncWithError {
	return func(w http.ResponseWriter, req *http.Request) error {
		req.Body = http.MaxBytesReader(w, req.Body, maxUploadSize)
		err := req.ParseMultipartForm(maxUploadSize)
		if err != nil {
			return fmt.Errorf("error parsing upload form: %s", err)
		}
		f, _, err := req.FormFile("notebook")
		defer f.Close()
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("error reading upload: %s", err)
		}
		// TODO(salikh): Add user identifier to the file name.
		filename := filepath.Join(s.UploadDir, uuid.New().String()+".ipynb")
		err = ioutil.WriteFile(filename, b, 0700)
		if err != nil {
			return fmt.Errorf("error writing uploaded file: %s", err)
		}
		err = s.scheduleCheck(filename)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "OK\n")
		return nil
	}
}

func (s *Server) scheduleCheck(filename string) error {
	fmt.Printf("TODO(salikh): Run checker for %q\n", filename)
	return nil
}

func (s *Server) uploadForm() httpHandleFuncWithError {
	return func(w http.ResponseWriter, req *http.Request) error {
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
