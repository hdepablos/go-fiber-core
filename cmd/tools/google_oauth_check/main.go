package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

func main() {
	_ = godotenv.Load()

	var expectedDomain string
	flag.StringVar(&expectedDomain, "expected-domain", "", "Dominio esperado, por ejemplo: empresa.com (opcional)")
	flag.Parse()

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	fmt.Println("== Google OAuth2 Check ==")
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		fmt.Println("ERROR: faltan variables de entorno requeridas:")
		if clientID == "" {
			fmt.Println("- GOOGLE_CLIENT_ID (vacío)")
		}
		if clientSecret == "" {
			fmt.Println("- GOOGLE_CLIENT_SECRET (vacío)")
		}
		if redirectURL == "" {
			fmt.Println("- GOOGLE_REDIRECT_URL (vacío)")
		}
		os.Exit(2)
	}

	u, err := url.Parse(redirectURL)
	if err != nil {
		fmt.Printf("ERROR: GOOGLE_REDIRECT_URL inválida: %v\n", err)
		os.Exit(2)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		fmt.Printf("ERROR: GOOGLE_REDIRECT_URL debe ser http/https, actual: %q\n", u.Scheme)
		os.Exit(2)
	}
	if u.Host == "" || u.Path == "" {
		fmt.Println("ERROR: GOOGLE_REDIRECT_URL debe incluir host y path, por ejemplo: http://localhost:9009/api/v1/oauth/google/callback")
		os.Exit(2)
	}

	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		fmt.Printf("ERROR: GOOGLE_REDIRECT_URL debe incluir puerto. Ejemplo: http://localhost:9009%s\n", u.Path)
		os.Exit(2)
	}
	if host == "" {
		host = "localhost"
	}

	state, err := randomState()
	if err != nil {
		fmt.Printf("ERROR: no se pudo generar state: %v\n", err)
		os.Exit(1)
	}

	oauthCfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	authURL := oauthCfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOnline,
		oauth2.SetAuthURLParam("include_granted_scopes", "true"),
	)

	fmt.Println("Paso 1: Abre este URL en tu navegador y completa el login con tu email corporativo:")
	fmt.Println(authURL)
	fmt.Println()
	fmt.Printf("Paso 2: Esperando callback en %s://%s%s\n", u.Scheme, u.Host, u.Path)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	var once sync.Once
	done := make(chan struct{})

	mux.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() {
			defer close(done)
		})

		q := r.URL.Query()
		if providerErr := q.Get("error"); providerErr != "" {
			http.Error(w, "Login cancelado o bloqueado por el provider. Revisa consola.", http.StatusUnauthorized)
			fmt.Printf("ERROR: provider devolvió error=%q\n", providerErr)
			return
		}

		code := q.Get("code")
		if code == "" {
			http.Error(w, "Falta code en callback. Revisa consola.", http.StatusBadRequest)
			fmt.Println("ERROR: callback sin code")
			return
		}

		gotState := q.Get("state")
		if gotState == "" || gotState != state {
			http.Error(w, "State inválido. Revisa consola.", http.StatusUnauthorized)
			fmt.Println("ERROR: state inválido o no coincide")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		fmt.Println("Paso 3: Intercambiando code por access_token...")
		tok, err := oauthCfg.Exchange(ctx, code)
		if err != nil {
			http.Error(w, "Exchange falló. Revisa consola.", http.StatusUnauthorized)
			fmt.Printf("ERROR: exchange falló: %v\n", err)
			return
		}
		if tok == nil || tok.AccessToken == "" {
			http.Error(w, "Exchange devolvió token vacío. Revisa consola.", http.StatusUnauthorized)
			fmt.Println("ERROR: token/access_token vacío")
			return
		}

		fmt.Println("Paso 4: Consultando userinfo...")
		info, err := fetchUserInfo(ctx, tok.AccessToken)
		if err != nil {
			http.Error(w, "UserInfo falló. Revisa consola.", http.StatusUnauthorized)
			fmt.Printf("ERROR: userinfo falló: %v\n", err)
			return
		}

		okDomain := true
		if expectedDomain != "" {
			okDomain = strings.HasSuffix(strings.ToLower(info.Email), "@"+strings.ToLower(expectedDomain))
		}

		fmt.Println("Paso 5: Resultado")
		fmt.Printf("- email: %s\n", info.Email)
		fmt.Printf("- verified_email: %v\n", info.VerifiedEmail)
		fmt.Printf("- domain check: %v\n", okDomain)
		fmt.Println("- userinfo (json):")
		b, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(b))
		fmt.Println()

		if !info.VerifiedEmail {
			fmt.Println("FAIL: Google devolvió verified_email=false. Ese usuario no debería autenticarse.")
		} else if expectedDomain != "" && !okDomain {
			fmt.Println("FAIL: el email no coincide con el dominio esperado.")
		} else {
			fmt.Println("OK: login con Google + userinfo funciona para este usuario.")
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("Listo. Puedes volver a la consola.\n"))
	})

	go func() {
		_ = srv.ListenAndServe()
	}()

	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func fetchUserInfo(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("userinfo respondió %d: %s", resp.StatusCode, string(body))
	}

	var info googleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	if info.Email == "" {
		return nil, errors.New("userinfo no devolvió email")
	}
	return &info, nil
}
