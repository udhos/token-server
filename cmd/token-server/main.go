// Package main implements the tool.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/udhos/boilerplate/boilerplate"
	"github.com/udhos/boilerplate/envconfig"
	"github.com/udhos/oauth2clientcredentials/clientcredentials"
)

const version = "1.0.9"

type application struct {
	clientCredentials bool
	expireSeconds     int
}

func main() {

	var showVersion bool
	flag.BoolVar(&showVersion, "version", showVersion, "show version")
	flag.Parse()

	me := filepath.Base(os.Args[0])

	{
		v := boilerplate.LongVersion(me + " version=" + version)
		if showVersion {
			fmt.Print(v)
			fmt.Println()
			return
		}
		log.Print(v)
	}

	env := envconfig.NewSimple(me)

	addr := env.String("ADDR", ":8080")
	pathToken := env.String("ROUTE", "/token")
	health := env.String("HEALTH", "/health")

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	app := &application{
		expireSeconds:     env.Int("EXPIRE_SECONDS", 600),
		clientCredentials: env.Bool("CLIENT_CREDENTIALS", false),
	}

	const root = "/"

	register(mux, addr, root, handlerRoot)
	register(mux, addr, health, handlerHealth)
	register(mux, addr, pathToken, func(w http.ResponseWriter, r *http.Request) { handlerToken(w, r, app) })

	go listenAndServe(server, addr)

	select {} // wait forever
}

func register(mux *http.ServeMux, addr, path string, handler http.HandlerFunc) {
	mux.HandleFunc(path, handler)
	log.Printf("registered on port %s path %s", addr, path)
}

func listenAndServe(s *http.Server, addr string) {
	log.Printf("listening on port %s", addr)
	err := s.ListenAndServe()
	log.Fatalf("listening on port %s: %v", addr, err)
}

// httpJSON replies to the request with the specified error message and HTTP code.
// It does not otherwise end the request; the caller should ensure no further
// writes are done to w.
// The message should be JSON.
func httpJSON(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, message)
}

func response(w http.ResponseWriter, r *http.Request, status int, message string) {
	hostname, errHost := os.Hostname()
	if errHost != nil {
		log.Printf("hostname error: %v", errHost)
	}
	reply := fmt.Sprintf(`{"message":"%s","status":"%d","path":"%s","method":"%s","host":"%s","serverHostname":"%s"}`,
		message, status, r.RequestURI, r.Method, r.Host, hostname)
	httpJSON(w, reply, status)
}

func handlerRoot(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s %s - 404 not found", r.RemoteAddr, r.Method, r.RequestURI)
	response(w, r, http.StatusNotFound, "not found")
}

func handlerHealth(w http.ResponseWriter, r *http.Request) {
	response(w, r, http.StatusOK, "health ok")
}

var sampleSecretKey = []byte("SecretYouShouldHide")

func handlerToken(w http.ResponseWriter, r *http.Request, app *application) {

	if app.clientCredentials {
		req, errReq := clientcredentials.DecodeRequestBody(r)
		if errReq != nil {
			log.Printf("%s %s %s - decode request body - 400 bad request: %v",
				r.RemoteAddr, r.Method, r.RequestURI, errReq)
			response(w, r, http.StatusBadRequest, "bad request")
			return
		}

		log.Printf("method=%s grant_type=%s client_id=%s client_secret=%s",
			r.Method, req.GrantType, req.ClientID, req.ClientSecret)

		if req.GrantType != "client_credentials" {
			log.Printf("%s %s %s - wrong grant type - 401 unauthorized", r.RemoteAddr, r.Method, r.RequestURI)
			response(w, r, http.StatusUnauthorized, "unauthorized")
			return
		}

		if req.ClientID != "admin" || req.ClientSecret != "admin" {
			log.Printf("%s %s %s - bad credentials - 401 unauthorized", r.RemoteAddr, r.Method, r.RequestURI)
			response(w, r, http.StatusUnauthorized, "unauthorized")
			return
		}
	}

	accessToken, errAccess := newToken(app.expireSeconds)
	if errAccess != nil {
		log.Printf("%s %s %s - access token - 500 server error: %v",
			r.RemoteAddr, r.Method, r.RequestURI, errAccess)
		response(w, r, http.StatusInternalServerError, "server error")
		return
	}

	var reply map[string]any
	var replyStr string

	if app.clientCredentials {
		replyStr = clientcredentials.EncodeResponseBody(accessToken, app.expireSeconds)

	} else {
		reply = map[string]any{
			"token":      accessToken,
			"token_type": "Bearer",
		}
		buf, errJSON := json.Marshal(reply)
		if errJSON != nil {
			log.Printf("%s %s %s - json error - 500 server error", r.RemoteAddr, r.Method, r.RequestURI)
			response(w, r, http.StatusInternalServerError, "server error")
			return
		}
		replyStr = string(buf)
	}

	log.Printf("%s %s %s - 200 ok", r.RemoteAddr, r.Method, r.RequestURI)

	httpJSON(w, replyStr, http.StatusOK)
}

func newToken(exp int) (string, error) {
	accessToken := jwt.New(jwt.SigningMethodHS256)
	claims := accessToken.Claims.(jwt.MapClaims)
	now := time.Now()
	claims["iat"] = now.Unix()
	if exp > 0 {
		claims["exp"] = now.Add(time.Duration(exp) * time.Second).Unix()
	}

	str, errSign := accessToken.SignedString(sampleSecretKey)
	if errSign != nil {
		return "", errSign
	}
	return str, nil
}
