package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/sessions"
	"gopkg.in/oauth2.v3/errors"
	"gopkg.in/oauth2.v3/manage"
	"gopkg.in/oauth2.v3/models"
	"gopkg.in/oauth2.v3/server"
	"gopkg.in/oauth2.v3/store"
)

var (
	port             = flag.Int("port", 9096, "The port to listen on.")
	hostname         = flag.String("hostname", "localhost", "The host name to listen on and use in the advertised URLs.")
	clientSecret     = flag.String("client_secret", "999999", "The client secret string.")
	clientID         = flag.String("client_id", "000000", "The client ID.")
	clientDomain     = flag.String("client_domain", "http://localhost:8080", "The client domain.")
	userName         = flag.String("user_name", "user", "The user name. Used both as ID and as a login name.")
	userPassword     = flag.String("user_password", "password", "The user password.")
	cookieAuthKey    = flag.String("cookie_auth_key", "abcd", "Cookie auth key.")
	cookieEncryptKey = flag.String("cookie_encrypt_key", "abcdef", "Cookie encrypt key.")
)

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

var (
	cookieStore *sessions.CookieStore
)

const (
	authSessionName = "auth"
	userKey         = "user"
	redirectKey     = "redirect_uri"
)

func main() {
	flag.Parse()
	cookieStore = sessions.NewCookieStore([]byte(*cookieAuthKey),
		[]byte(*cookieEncryptKey))
	manager := manage.NewDefaultManager()
	// token memory store
	manager.MustTokenStorage(store.NewMemoryTokenStore())

	// client memory store
	clientStore := store.NewClientStore()
	clientStore.Set(*userName, &models.Client{
		ID:     *clientID,
		Secret: *clientSecret,
		Domain: *clientDomain,
	})
	manager.MapClientStorage(clientStore)

	srv := server.NewDefaultServer(manager)
	srv.SetPasswordAuthorizationHandler(func(username, password string) (userID string, err error) {
		if username == *userName && password == *userPassword {
			userID = *userName
		}
		return
	})

	srv.SetUserAuthorizationHandler(func(w http.ResponseWriter, req *http.Request) (userID string, err error) {
		session, err := cookieStore.Get(req, authSessionName)
		if err != nil {
			err = fmt.Errorf("cookie store error: %s", err)
			return
		}
		user, ok := session.Values[userKey].(string)
		if ok && user != "" {
			delete(session.Values, userKey)
			session.Save(req, w)
			userID = user
			return
		}
		req.ParseForm()
		session.Values[redirectKey] = req.Form.Encode()
		session.Save(req, w)
		http.Redirect(w, req, "/login", http.StatusFound)
		return
	})

	srv.SetAllowGetAccessRequest(true)
	srv.SetClientInfoHandler(server.ClientFormHandler)

	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		log.Println("Internal Error:", err.Error())
		return
	})

	srv.SetResponseErrorHandler(func(re *errors.Response) {
		log.Println("Response Error:", re.Error.Error())
	})

	http.HandleFunc("/authorize", func(w http.ResponseWriter, req *http.Request) {
		req.ParseForm()
		session, err := cookieStore.Get(req, authSessionName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		redirectForm, ok := session.Values[redirectKey].(string)
		if ok && redirectForm != "" {
			form, err := url.ParseQuery(redirectForm)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			req.Form = form
		}
		log.Printf("%s %s %s\n", req.Method, req.URL.Path, req.Form)
		log.Printf("session: %#v", session.Values)
		err = srv.HandleAuthorizeRequest(w, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	http.HandleFunc("/token", func(w http.ResponseWriter, req *http.Request) {
		log.Printf("%s %s\n", req.Method, req.URL.Path)
		srv.HandleTokenRequest(w, req)
	})

	http.HandleFunc("/userinfo", func(w http.ResponseWriter, req *http.Request) {
		err := srv.HandleAuthorizeRequest(w, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		userinfo := &UserInfo{
			ID:    "testID",
			Email: "test@test.com",
		}
		b, err := json.Marshal(userinfo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, req *http.Request) {
		session, err := cookieStore.Get(req, authSessionName)
		if err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, err.Error())
			return
		}
		if req.Method == "POST" {
			user := req.FormValue("username")
			password := req.FormValue("password")
			if user != *userName || password != *userPassword {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintf(w, "Login error, <a href='/login'>Try again</a>. User: %q, password: %q", user, password)
				return
			}
			session.Values[userKey] = user
			session.Save(req, w)
			redirectForm, ok := session.Values[redirectKey].(string)
			if ok && redirectForm != "" {
				http.Redirect(w, req, "/authorize", http.StatusTemporaryRedirect)
				return
			}
			fmt.Fprintf(w, `<pre>session: %#v</pre>`, session.Values)
			fmt.Fprintf(w, "login OK")
			return
		}
		user, ok := session.Values[userKey]
		if ok && user != "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<pre>session: %#v</pre>`, session.Values)
			fmt.Fprintf(w, `Already logged in.`)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<pre>session: %#v</pre>`, session.Values)
		fmt.Fprintf(w, `<form action="/login" method="POST">
<label for="username">User name
<input type="text" name="username" value="user"></label>
<br>
<label for="password">Password
<input type="password" name="password" value="password"></label>
<input type="submit" value="Login">
</form>`)
	})

	http.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		url := fmt.Sprintf("http://%s:%d", *hostname, *port)
		fmt.Fprintf(w, `{"issuer": "%s",
"authorization_endpoint": "%s",
"token_endpoint": "%s",
"userinfo_endpoint": "%s/userinfo",
"response_types_supported": ["code"],
"subject_types_supported": ["public"],
"scopes_supported": ["openid", "email", "profile"],
"token_endpoint_auth_methods_supported": ["client_secret_post"],
"claims_supported": ["email"],
"code_challenge_methods_supported": ["plain"]
}`, url, url, url, url)
	})

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("\n  OAuth server listening on http://localhost%s\n\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}

}
