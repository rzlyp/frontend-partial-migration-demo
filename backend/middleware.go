package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey int

const ctxUser ctxKey = 1

func userFrom(r *http.Request) *User { return r.Context().Value(ctxUser).(*User) }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// authn accepts the access JWT from the HttpOnly cookie (browsers) or an Authorization header (scripts, curl).
func (s *Server) authn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := bearer(r)
		if tok == "" {
			if c, err := r.Cookie(cookieSession); err == nil {
				tok = c.Value
			}
		}
		if tok == "" {
			writeErr(w, http.StatusUnauthorized, "authentication required")
			return
		}
		claims := &Claims{}
		_, err := jwt.ParseWithClaims(tok, claims,
			func(*jwt.Token) (any, error) { return s.cfg.JWTSecret, nil },
			jwt.WithValidMethods([]string{"HS256"}), // blocks alg=none / algorithm confusion
			jwt.WithIssuer(jwtIssuer),
			jwt.WithExpirationRequired(),
		)
		u := s.usersByID[claims.Subject]
		if err != nil || u == nil {
			writeErr(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxUser, u)))
	})
}

// csrf: double-submit token for cookie-authenticated, state-changing requests.
// Requests that carry an Authorization header are not ambient-credential requests, so they are exempt.
func (s *Server) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if bearer(r) == "" {
				c, err := r.Cookie(cookieCSRF)
				h := r.Header.Get("X-CSRF-Token")
				if err != nil || c.Value == "" || subtle.ConstantTimeCompare([]byte(c.Value), []byte(h)) != 1 {
					writeErr(w, http.StatusForbidden, "CSRF token missing or invalid")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requirePerm(perm string, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hasPerm(userFrom(r).Role, perm) {
			writeErr(w, http.StatusForbidden, "missing permission: "+perm)
			return
		}
		next(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(c int) { w.status = c; w.ResponseWriter.WriteHeader(c) }

func withBasics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}
