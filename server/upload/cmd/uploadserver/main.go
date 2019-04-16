package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/google/prog-edu-assistant/uploadserver"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	hostname    = flag.String("hostname", "localhost", "The host name to use for callbacks.")
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
	openIDIssuer = flag.String("openid_issuer", "https://accounts.google.com",
		"The URL of the OpenID Connect issuer. /.well-known/openid-configuration will be "+
			"requested for detailed endpoint configuration. If empty, defaults to Google.")
	useOpenID = flag.Bool("use_openid", false, "If true, use OpenID Connect authentication"+
		" provided by the issuer specified with --openid_issuer.")
	allowedUsersFile = flag.String("allowed_users_file", "allowed_users.txt", "The file name of a text file "+
		"with one user email per line.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	endpoint := google.Endpoint
	var userinfoEndpoint string
	if *openIDIssuer != "" {
		wellKnownURL := *openIDIssuer + "/.well-known/openid-configuration"
		resp, err := http.Get(wellKnownURL)
		if err != nil {
			return fmt.Errorf("Error on GET %s: %s", wellKnownURL, err)
		}
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		data := make(map[string]interface{})
		err = json.Unmarshal(b, &data)
		if err != nil {
			return fmt.Errorf("Error parsing response from %s: %s", wellKnownURL, err)
		}
		// Override the authentication endpoint.
		auth_ep, ok := data["authorization_endpoint"].(string)
		if !ok {
			return fmt.Errorf("response from %s does not have 'authorization_endpoint' key", wellKnownURL)
		}
		token_ep, ok := data["token_endpoint"].(string)
		if !ok {
			return fmt.Errorf("response from %s does not have 'token_endpoint' key", wellKnownURL)
		}
		endpoint = oauth2.Endpoint{
			AuthURL:   auth_ep,
			TokenURL:  token_ep,
			AuthStyle: oauth2.AuthStyleInParams,
		}
		log.Infof("auth endpoint: %#v", endpoint)
		userinfoEndpoint, ok = data["userinfo_endpoint"].(string)
		if !ok {
			return fmt.Errorf("response from %s does not have 'userinfo_endpoint' key", wellKnownURL)
		}
		log.Infof("userinfo endpoint: %#v", userinfoEndpoint)
	}
	allowedUsers := make(map[string]bool)
	if *allowedUsersFile != "" {
		b, err := ioutil.ReadFile(*allowedUsersFile)
		if err != nil {
			return fmt.Errorf("error reading --allowed_users_file %q: %s", *allowedUsersFile, err)
		}
		for _, email := range strings.Split(string(b), "\n") {
			if email == "" {
				continue
			}
			allowedUsers[email] = true
		}
	}
	addr := ":" + strconv.Itoa(*port)
	protocol := "http"
	if *useHTTPS {
		protocol = "https"
	}
	serverURL := fmt.Sprintf("%s://%s%s", protocol, *hostname, addr)
	s := uploadserver.New(uploadserver.Options{
		ServerURL:        serverURL,
		UploadDir:        *uploadDir,
		DisableCORS:      *disableCORS,
		UseOpenID:        *useOpenID,
		AllowedUsers:     allowedUsers,
		AuthEndpoint:     endpoint,
		UserinfoEndpoint: userinfoEndpoint,
		ClientID:         os.Getenv("CLIENT_ID"),
		ClientSecret:     os.Getenv("CLIENT_SECRET"),
		CookieAuthKey:    os.Getenv("COOKIE_AUTH_KEY"),
		CookieEncryptKey: os.Getenv("COOKIE_ENCRYPT_KEY"),
	})
	fmt.Printf("\n  Serving on %s\n\n", serverURL)
	if *useHTTPS {
		return s.ListenAndServeTLS(addr, *sslCertFile, *sslKeyFile)
	}
	return s.ListenAndServe(addr)
}
