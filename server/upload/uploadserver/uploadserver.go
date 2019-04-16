package uploadserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	log "github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

type Options struct {
	// The base URL of this server. This is used to construct callback URL.
	ServerURL string
	// UploadDir specifies the directory to write uploaded files to
	// and to serve on /uploads.
	UploadDir string
	// DisableCORS specifies whether the server should disable CORS
	// (Cross-origin request sharing) checks in browser by adding
	// Access-Control-Allow-Origin:* HTTP header.
	DisableCORS bool
	// UseOpenID enables authentication using OpenID Connect.
	UseOpenID bool
	// AllowedUsers lists the users that are authorized to use this service.
	AllowedUsers map[string]bool
	// AuthEndpoint specifies the OpenID Connect authentication and token endpoints.
	AuthEndpoint oauth2.Endpoint
	// UserinfoEndpoint specifies the user info endpoint.
	UserinfoEndpoint string
	// ClientID is used for OpenID Connect authentication.
	ClientID string
	// ClientSecret is used for OpenID Connect authentication.
	ClientSecret string
	// Set to 32 or 64 random bytes.
	CookieAuthKey string
	// Set to 16, 24 or 32 random bytes.
	CookieEncryptKey string
}

type Server struct {
	opts        Options
	mux         *http.ServeMux
	cookieStore *sessions.CookieStore
	// OauthConfig specifies endpoing configuration for the OpenID Connect
	// authentication.
	oauthConfig *oauth2.Config
	// A random value used to match authentication callback to the request.
	oauthState string
}

const UserSessionName = "user_session"

func New(opts Options) *Server {
	mux := http.NewServeMux()
	s := &Server{
		opts:        opts,
		mux:         mux,
		cookieStore: sessions.NewCookieStore([]byte(opts.CookieAuthKey), []byte(opts.CookieEncryptKey)),
		oauthConfig: &oauth2.Config{
			RedirectURL:  opts.ServerURL + "/callback",
			ClientID:     opts.ClientID,
			ClientSecret: opts.ClientSecret,
			Scopes:       []string{"profile", "email", "openid"},
			Endpoint:     opts.AuthEndpoint,
		},
		oauthState: uuid.New().String(),
	}
	mux.Handle("/", handleError(s.uploadForm))
	mux.Handle("/upload", handleError(s.handleUpload))
	mux.Handle("/uploads", http.StripPrefix("/uploads",
		http.FileServer(http.Dir(s.opts.UploadDir))))
	if s.opts.UseOpenID {
		mux.Handle("/login", handleError(s.handleLogin))
		mux.Handle("/callback", handleError(s.handleCallback))
		mux.Handle("/logout", handleError(s.handleLogout))
	}
	return s
}

func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	return http.ListenAndServeTLS(addr, certFile, keyFile, s.mux)
}

type httpError int

func (e httpError) Error() string {
	return http.StatusText(int(e))
}

func handleError(fn func(http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := fn(w, req)
		if err != nil {
			log.Error(err.Error())
			status, ok := err.(httpError)
			if ok {
				http.Error(w, err.Error(), int(status))
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, req *http.Request) error {
	url := s.oauthConfig.AuthCodeURL(s.oauthState)
	http.Redirect(w, req, url, http.StatusTemporaryRedirect)
	return nil
}

func (s *Server) getUserInfo(state string, code string) ([]byte, error) {
	if state != s.oauthState {
		return nil, fmt.Errorf("invalid oauth state")
	}
	token, err := s.oauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err)
	}
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting user info: %s", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading user info response: %s", err)
	}
	return b, nil
}

type UserProfile struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Link          string `json:"link"`
	Picture       string `json:"picture"`
}

func (s *Server) handleCallback(w http.ResponseWriter, req *http.Request) error {
	req.ParseForm()
	b, err := s.getUserInfo(req.FormValue("state"), req.FormValue("code"))
	if err != nil {
		return err
	}
	var profile UserProfile
	//fmt.Printf("%s\n", b)
	err = json.Unmarshal(b, &profile)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	session, err := s.cookieStore.Get(req, UserSessionName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}
	if !s.opts.AllowedUsers[profile.Email] {
		delete(session.Values, "email")
		session.Save(req, w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(fmt.Sprintf("<title>Forbidden</title>User %s is not authorized.<br>"+
			"Try a different Google account. <a href='https://mail.google.com/mail/logout'>Log out of Google</a>.", profile.Email)))
		return nil
	}
	session.Values["email"] = profile.Email
	session.Save(req, w)
	w.Write([]byte(fmt.Sprintf("<title>Welcome</title>Welcome, %s.<br><a href='/'>Go to top</a>.", profile.Email)))
	return nil
}

func (s *Server) handleLogout(w http.ResponseWriter, req *http.Request) error {
	session, err := s.cookieStore.Get(req, UserSessionName)
	if err != nil {
		return err
	}
	delete(session.Values, "email")
	session.Save(req, w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("Logged out"))
	return nil
}

func (s *Server) authenticate(w http.ResponseWriter, req *http.Request) error {
	session, err := s.cookieStore.Get(req, UserSessionName)
	if err != nil {
		return err
	}
	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		return httpError(http.StatusUnauthorized)
	}
	if !s.opts.AllowedUsers[email] {
		return httpError(http.StatusForbidden)
	}
	return nil
}

const maxUploadSize = 1048576

func (s *Server) handleUpload(w http.ResponseWriter, req *http.Request) error {
	if s.opts.UseOpenID {
		err := s.authenticate(w, req)
		if err != nil {
			return err
		}
	}
	if s.opts.DisableCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
	}
	if req.Method == "OPTIONS" {
		log.Infof("OPTIONS %s", req.URL.Path)
		return nil
	}
	if req.Method != "POST" {
		return fmt.Errorf("Unsupported method %s on %s", req.Method, req.URL.Path)
	}
	log.Infof("POST %s", req.URL.Path)
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
	filename := filepath.Join(s.opts.UploadDir, uuid.New().String()+".ipynb")
	err = ioutil.WriteFile(filename, b, 0700)
	if err != nil {
		return fmt.Errorf("error writing uploaded file: %s", err)
	}
	err = s.scheduleCheck(filename)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	fmt.Fprintf(w, "OK\n")
	return nil
}

func (s *Server) scheduleCheck(filename string) error {
	fmt.Printf("TODO(salikh): Run checker for %q\n", filename)
	return nil
}

func (s *Server) uploadForm(w http.ResponseWriter, req *http.Request) error {
	if s.opts.UseOpenID {
		err := s.authenticate(w, req)
		if err != nil {
			return err
		}
	}
	if req.Method != "GET" {
		return fmt.Errorf("Unsupported method %s on %s", req.Method, req.URL.Path)
	}
	log.Infof("GET %s", req.URL.Path)
	_, err := w.Write([]byte(uploadHTML))
	return err
}

const uploadHTML = `<!DOCTYPE html>
<title>Upload form</title>
<form method="POST" action="/upload" enctype="multipart/form-data">
	<input type="file" name="notebook">
	<input type="submit" value="Upload">
</form>`
