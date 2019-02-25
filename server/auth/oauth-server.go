package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"gopkg.in/oauth2.v3/errors"
	"gopkg.in/oauth2.v3/manage"
	"gopkg.in/oauth2.v3/models"
	"gopkg.in/oauth2.v3/server"
	"gopkg.in/oauth2.v3/store"
)

var (
	port         = flag.Int("port", 9096, "The port to listen on.")
	hostname     = flag.String("hostname", "localhost", "The host name to listen on and use in the advertised URLs.")
	clientSecret = flag.String("client_secret", "999999", "The client secret string.")
	clientID     = flag.String("client_id", "000000", "The client ID.")
	clientDomain = flag.String("client_domain", "http://localhost:8080", "The client domain.")
	userID       = flag.String("user_id", "user", "The user ID.")
)

func main() {
	flag.Parse()
	manager := manage.NewDefaultManager()
	// token memory store
	manager.MustTokenStorage(store.NewMemoryTokenStore())

	// client memory store
	clientStore := store.NewClientStore()
	clientStore.Set(*userID, &models.Client{
		ID:     *clientID,
		Secret: *clientSecret,
		Domain: *clientDomain,
	})
	manager.MapClientStorage(clientStore)

	srv := server.NewDefaultServer(manager)
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
		log.Printf("%s %s\n", req.Method, req.URL.Path)
		err := srv.HandleAuthorizeRequest(w, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	http.HandleFunc("/token", func(w http.ResponseWriter, req *http.Request) {
		log.Printf("%s %s\n", req.Method, req.URL.Path)
		srv.HandleTokenRequest(w, req)
	})

	http.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		url := fmt.Sprintf("http://%s:%d", *hostname, *port)
		fmt.Fprintf(w, `{"issuer": "%s",
"authorization_endpoint": "%s/auth",
"token_endpoint": "%s/token",
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
