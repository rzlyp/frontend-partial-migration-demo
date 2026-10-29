package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Config comes from the environment so the same image runs on a laptop, staging and production.
type Config struct {
	Port         string
	JWTSecret    []byte
	AccessTTL    time.Duration // lifetime of the access JWT (kept short)
	RefreshTTL   time.Duration // lifetime of the refresh token (sliding: renewed on every rotation)
	RefreshGrace time.Duration // an already-rotated refresh token is still honoured this long (concurrent tabs/apps)
	CookieSecure bool          // true behind HTTPS (ALB/CloudFront)
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDur(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		log.Printf("invalid %s=%q, using %s", key, v, def)
	}
	return def
}

func loadConfig() Config {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		log.Println("WARNING: JWT_SECRET missing or shorter than 32 bytes; using an insecure development secret")
		secret = "insecure-development-secret-change-me!!"
	}
	secure, _ := strconv.ParseBool(env("COOKIE_SECURE", "false"))
	return Config{
		Port:         env("PORT", "8080"),
		JWTSecret:    []byte(secret),
		AccessTTL:    envDur("ACCESS_TTL", 15*time.Minute),
		RefreshTTL:   envDur("REFRESH_TTL", 8*time.Hour),
		RefreshGrace: envDur("REFRESH_GRACE", 30*time.Second),
		CookieSecure: secure,
	}
}

func main() {
	cfg := loadConfig()
	srv, err := NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}
	httpSrv := &http.Server{Addr: ":" + cfg.Port, Handler: srv.Routes(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("pos-backend listening on :%s (access ttl %s, refresh ttl %s, grace %s)",
		cfg.Port, cfg.AccessTTL, cfg.RefreshTTL, cfg.RefreshGrace)
	log.Fatal(httpSrv.ListenAndServe())
}
