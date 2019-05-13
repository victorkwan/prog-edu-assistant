package uploadserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/google/prog-edu-assistant/queue"
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
	// QueueName is the name of the queue to post uploads.
	QueueName string
	// Channel is the interface to the message queue.
	*queue.Channel
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
	opts            Options
	mux             *http.ServeMux
	reportTimestamp map[string]time.Time
	cookieStore *sessions.CookieStore
	// OauthConfig specifies endpoing configuration for the OpenID Connect
	// authentication.
	oauthConfig *oauth2.Config
	// A random value used to match authentication callback to the request.
	oauthState string
}

func New(opts Options) *Server {
	mux := http.NewServeMux()
	s := &Server{
		opts:            opts,
		mux:             mux,
		reportTimestamp: make(map[string]time.Time),
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
	mux.Handle("/uploads/", http.StripPrefix("/uploads",
		http.FileServer(http.Dir(s.opts.UploadDir))))
	mux.Handle("/report/", handleError(s.handleReport))
	if s.opts.UseOpenID {
		mux.Handle("/login", handleError(s.handleLogin))
		mux.Handle("/callback", handleError(s.handleCallback))
		mux.Handle("/logout", handleError(s.handleLogout))
	}
	return s
}

const UserSessionName = "user_session"

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
			glog.Error(err.Error())
			status, ok := err.(httpError)
			if ok {
				http.Error(w, err.Error(), int(status))
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	})
}

func (s *Server) handleReport(w http.ResponseWriter, req *http.Request) error {
	if s.opts.DisableCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
	}
	basename := path.Base(req.URL.Path)
	filename := filepath.Join(s.opts.UploadDir, basename+".txt")
	glog.V(5).Infof("checking %q for existence", filename)
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		// Serve a placeholder autoreload page.
		reloadMs := int64(500)
		ts := s.reportTimestamp[basename]
		if ts.IsZero() {
			// Store the first request time
			s.reportTimestamp[basename] = time.Now()
			// TODO(salikh): Eventually clean up old entries from reportTimestamp map.
		} else {
			// Back off automatically.
			reloadMs = time.Since(ts).Nanoseconds() / 1000000
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// TODO(salikh): Make timeout configurable.
		if reloadMs > 10000 {
			// Reset for retry.
			reloadMs = 500
			s.reportTimestamp[basename] = time.Now()
		}
		if reloadMs > 5000 {
			fmt.Fprintf(w, `<title>Something weng wrong</title>
<h2>Error</h2>
Something went wrong, please retry your upload.
`)
			return nil
		}
		fmt.Fprintf(w, `<title>Please wait</title>
<script>
function refresh(t) {
	setTimeout("location.reload(true)", t)
}
</script>
<body onload="refresh(%d)">
<h2>Waiting for %d seconds, report is being generated now</h2>
</body>`, reloadMs, (reloadMs+999)/1000)
		return nil
	}
	if err != nil {
		return err
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	data := make(map[string]interface{})
	err = json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<title>Report for %s</title>`, basename)
	if v, ok := data["reports"]; ok {
		reports, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected reports to be map[string]interface{}, got %s", reflect.TypeOf(v))
		}
		// Just concatenate all reports.
		for exercise_id, report := range reports {
			fmt.Fprintf(w, "<h2>%s</h2>", exercise_id)
			html, ok := report.(string)
			if !ok {
				return fmt.Errorf("expected report to be a string, got %s", reflect.TypeOf(report))
			}
			fmt.Fprint(w, html)
		}
	}
	return nil
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
	if s.opts.DisableCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
	}
	if req.Method == "OPTIONS" {
		glog.Infof("OPTIONS %s", req.URL.Path)
		return nil
	}
	if req.Method != "POST" {
		return fmt.Errorf("Unsupported method %s on %s", req.Method, req.URL.Path)
	}
	if s.opts.UseOpenID {
		err := s.authenticate(w, req)
		if err != nil {
			return err
		}
	}
	if req.Method != "POST" {
		return fmt.Errorf("Unsupported method %s on %s", req.Method, req.URL.Path)
	}
	glog.Infof("POST %s", req.URL.Path)
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
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	glog.V(5).Infof("Uploaded: %s", string(b))
	fmt.Fprintf(w, "/report/"+submissionID)
	return nil
}

func (s *Server) scheduleCheck(content []byte) error {
	return s.opts.Channel.Post(s.opts.QueueName, content)
}

func (s *Server) ListenForReports(ch <-chan []byte) {
	for b := range ch {
		glog.V(3).Infof("Received %d byte report", len(b))
		glog.V(5).Infof("Received: %s", string(b))
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
		filename := filepath.Join(s.opts.UploadDir, submissionID+".txt")
		err = ioutil.WriteFile(filename, b, 0775)
		if err != nil {
			glog.Errorf("Error writing to %q: %s", filename, err)
			continue
		}
	}
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
	glog.Infof("GET %s", req.URL.Path)
	_, err := w.Write([]byte(uploadHTML))
	return err
}

const uploadHTML = `<!DOCTYPE html>
<title>Upload form</title>
<form method="POST" action="/upload" enctype="multipart/form-data">
	<input type="file" name="notebook">
	<input type="submit" value="Upload">
</form>`
