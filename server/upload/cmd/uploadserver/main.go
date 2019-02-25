package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/google/prog-edu-assistant/uploadserver"
)

var (
	port        = flag.Int("port", 8080, "The port to serve HTTP/S.")
	hostname    = flag.String("hostname", "localhost", "The hostname of the server.")
	useHTTPS    = flag.Bool("use_https", false, "If true, use HTTPS instead of HTTP.")
	sslCertFile = flag.String("ssl_cert_file", "localhost.crt",
		"The path to the signed SSL server certificate.")
	sslKeyFile = flag.String("ssl_key_file", "localhost.key",
		"The path to the SSL server key.")
	uploadDir  = flag.String("upload_dir", "uploads", "The directory to write uploaded notebooks.")
	authServer = flag.String("auth_server", "http://localhost:9096",
		"The base URL of the OpenID Connect authentication server.")
	clientID = flag.String("client_id", "000000",
		"The client ID for OAuth2")
	clientSecret = flag.String("client_secret", "999999",
		"The client secret for OAuth2")
	cookieAuthKey = flag.String("cookie_auth", "abcdef",
		"The cookie auth key.")
	cookieEncryptKey = flag.String("cookie_encrypt_key", "abcdefgh",
		"The cookie encryption key.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

// Config contains the OpenID Connect server configuration that
// we care about.
// TODO(salikh): Make explicit the other assumptions we make on the auth
// server capabilities and check them against the configuration object.
type Config struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

func getConfig(baseURL string) (*Config, error) {
	resp, err := http.Get(baseURL + "/.well-known/openid-configuration")
	if err != nil {
		return nil, fmt.Errorf("error getting OpenID configuration: %s", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading OpenID configuration: %s", err)
	}
	var config Config
	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing OpenID configuration: %s", err)
	}
	log.Printf("OpenID Connect config: %#v", config)
	return &config, nil
}

func run() error {
	config, err := getConfig(*authServer)
	if err != nil {
		return err
	}
	protocol := "http"
	if *useHTTPS {
		protocol = "https"
	}
	s := server.New(server.Options{
		UploadDir:        *uploadDir,
		Protocol:         protocol,
		Hostname:         *hostname,
		Port:             *port,
		AuthorizeURL:     config.AuthorizationEndpoint,
		TokenURL:         config.TokenEndpoint,
		UserinfoURL:      config.UserinfoEndpoint,
		ClientID:         *clientID,
		ClientSecret:     *clientSecret,
		CookieAuthKey:    *cookieAuthKey,
		CookieEncryptKey: *cookieEncryptKey,
	})
	fmt.Printf("\n  Serving on %s://%s:%d\n\n", protocol, *hostname, *port)
	if *useHTTPS {
		return s.ListenAndServeTLS(*sslCertFile, *sslKeyFile)
	}
	return s.ListenAndServe()
}
