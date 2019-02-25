package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

type Options struct {
	UploadDir string
	// Protocol is either "http" or "https".
	Protocol string
	// Hostname is the hostname to listen on and to use in redirect URLs.
	Hostname string
	// Port is the port to listen on and to use in redirect URLs.
	Port int
	// AuthorizeURL is the URL of the OAuth2 authentication handler.
	AuthorizeURL string
	// TokenURL is the URL of the OAuth2 token handler.
	TokenURL string
	// UserinfoURL is the URL to obtain the user info.
	UserinfoURL string
	// ClientID is the client ID used by OAuth2. It must be obtained
	// from the auth provider in advance.
	ClientID string
	// ClientSecret is the client secret used by OAuth2. It must be obtained
	// from the auth provider in advance.
	ClientSecret string
	// CookieAuthKey is the auth key used by the cookie store for session
	// authentication. It should be a reasonably long random string (e.g. 32
	// bytes from /dev/urandom in hex encoding). Make sure the value
	// is secret and not submitted with the source code.
	CookieAuthKey string
	// CookieEncryptKey is the encryption key used by the cookie store.
	// Similar to CookieAuthKey, it should be long, random and secret string.
	CookieEncryptKey string
}

type Server struct {
	Options
	*mux.Router
	*sessions.CookieStore
	OauthConfig *oauth2.Config
}

func New(opts Options) *Server {
	r := mux.NewRouter()
	callbackURL := fmt.Sprintf("%s://%s:%d/callback",
		opts.Protocol, opts.Hostname, opts.Port)
	s := &Server{
		Options: opts,
		Router:  r,
		OauthConfig: &oauth2.Config{
			ClientID:     opts.ClientID,
			ClientSecret: opts.ClientSecret,
			// TODO(salikh): Confirm if this is the right setting.
			Scopes:      []string{"openid email profile"},
			RedirectURL: callbackURL,
			Endpoint: oauth2.Endpoint{
				AuthURL:  opts.AuthorizeURL + "/authorize",
				TokenURL: opts.TokenURL + "/token",
			},
		},
		CookieStore: sessions.NewCookieStore([]byte(opts.CookieAuthKey), []byte(opts.CookieEncryptKey)),
	}
	r.Handle("/upload", handleError(s.handleUpload())).Methods("POST")
	r.Handle("/", handleError(s.uploadForm())).Methods("GET")
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads",
		http.FileServer(http.Dir(s.Options.UploadDir))))
	r.HandleFunc("/upload", s.handleOptions()).Methods("OPTIONS")
	r.Handle("/login", handleError(s.handleLogin()))
	r.Handle("/callback", handleError(s.handleCallback()))
	r.Use(s.authenticate)
	return s
}

var (
	unauthenticatedPaths = map[string]bool{
		"/login":    true,
		"/callback": true,
	}
)

const userSessionName = "session"

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Printf("%s %s", req.Method, req.URL.Path)
		if unauthenticatedPaths[req.URL.Path] {
			next.ServeHTTP(w, req)
			return
		}
		// Other pages require authentication.
		session, err := s.CookieStore.Get(req, userSessionName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		email, ok := session.Values["email"].(string)
		if !ok || email == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("<title>Logged out</title>You are logged out.<br><a href='/login'>Log in</a>."))
			return
		}
		next.ServeHTTP(w, req)
	})
}

func (s *Server) listenAddr() string {
	return fmt.Sprintf("%s:%d", s.Options.Hostname, s.Options.Port)
}

func (s *Server) ListenAndServe() error {
	addr := s.listenAddr()
	return http.ListenAndServe(addr, s.Router)
}

func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	addr := s.listenAddr()
	return http.ListenAndServeTLS(addr, certFile, keyFile, s.Router)
}

type httpHandlerWithErr = func(http.ResponseWriter, *http.Request) error

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

var oauthStateString = "2@p8xdj5sdfkljvl23dfv"

func (s *Server) handleLogin() func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, req *http.Request) error {
		// TODO(salikh): randomize state string.
		url := s.OauthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, req, url, http.StatusTemporaryRedirect)
		return nil
	}
}

type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Link          string `json:"link"`
	Picture       string `json:"picture"`
}

func (s *Server) getUserInfo(state string, code string) (*UserInfo, error) {
	if state != oauthStateString {
		return nil, fmt.Errorf("invalid oauth state")
	}
	token, err := s.OauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange error: %s", err)
	}
	resp, err := http.Get(s.Options.UserinfoURL + "?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting user info: %s", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading user info response: %s", err)
	}
	var userinfo UserInfo
	err = json.Unmarshal(b, &userinfo)
	if err != nil {
		return nil, fmt.Errorf("error decoding userinfo: %s", err)
	}
	return &userinfo, nil
}

func (s *Server) handleCallback() httpHandleFuncWithError {
	return func(w http.ResponseWriter, req *http.Request) error {
		userinfo, err := s.getUserInfo(req.FormValue("state"), req.FormValue("code"))
		if err != nil {
			return fmt.Errorf("error in authentication callback: %s", err)
		}
		session, err := s.CookieStore.Get(req, userSessionName)
		if err != nil {
			return fmt.Errorf("error getting session: %s", err)
		}
		if userinfo.Email == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "<title>Forbidden</title>Invalid empty email. Try a different account.")
			return nil
		}
		session.Values["email"] = userinfo.Email
		session.Save(req, w)
		fmt.Fprintf(w, "Authenticated: %s", userinfo.Email)
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
