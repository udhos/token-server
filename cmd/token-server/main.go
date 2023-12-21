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
	"runtime"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/udhos/boilerplate/envconfig"
	_ "go.uber.org/automaxprocs"
)

const version = "0.1.1"

func getVersion(me string) string {
	return fmt.Sprintf("%s version=%s runtime=%s GOOS=%s GOARCH=%s GOMAXPROCS=%d",
		me, version, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0))
}

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
		v := getVersion(me)
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

	<-chan struct{}(nil)
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

func getParam(r *http.Request, key string) string {
	v := r.Form[key]
	if v == nil {
		return ""
	}
	return v[0]
}

func handlerToken(w http.ResponseWriter, r *http.Request, app *application) {

	if app.clientCredentials {
		if err := r.ParseForm(); err != nil {
			log.Printf("handlerToken: ParseForm: err: %v", err)
		}

		grantType := getParam(r, "grant_type")
		clientID := getParam(r, "client_id")
		clientSecret := getParam(r, "client_secret")

		log.Printf("method=%s grant_type=%s client_id=%s client_secret=%s",
			r.Method, grantType, clientID, clientSecret)

		if grantType != "client_credentials" {
			log.Printf("%s %s %s - wrong grant type - 401 unauthorized", r.RemoteAddr, r.Method, r.RequestURI)
			response(w, r, http.StatusUnauthorized, "unauthorized")
			return
		}

		if clientID != "admin" || clientSecret != "admin" {
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

	var reply map[string]interface{}

	if app.clientCredentials {
		refreshToken, errRefresh := newToken(0)
		if errRefresh != nil {
			log.Printf("%s %s %s - refresh token - 500 server error: %v",
				r.RemoteAddr, r.Method, r.RequestURI, errRefresh)
			response(w, r, http.StatusInternalServerError, "server error")
			return
		}

		reply = map[string]interface{}{
			"access_token":  accessToken,
			"token_type":    "Bearer",
			"refresh_token": refreshToken,
			"expires_in":    app.expireSeconds,
		}
	} else {
		reply = map[string]interface{}{
			"token":      accessToken,
			"token_type": "Bearer",
		}
	}

	buf, errJSON := json.Marshal(reply)
	if errJSON != nil {
		log.Printf("%s %s %s - json error - 500 server error", r.RemoteAddr, r.Method, r.RequestURI)
		response(w, r, http.StatusInternalServerError, "server error")
		return
	}

	log.Printf("%s %s %s - 200 ok", r.RemoteAddr, r.Method, r.RequestURI)

	httpJSON(w, string(buf), http.StatusOK)
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
