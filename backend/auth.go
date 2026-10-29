package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	cookieSession = "pos_session" // access JWT   - HttpOnly, Path=/
	cookieRefresh = "pos_refresh" // refresh token - HttpOnly, Path=/api/auth (only sent to auth endpoints)
	cookieCSRF    = "pos_csrf"    // CSRF token    - readable by JS, echoed in X-CSRF-Token
	jwtIssuer     = "pos-backend"
)

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"` // informational only; authorization uses the live user record
	jwt.RegisteredClaims
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Server) signAccess(u *User) (string, error) {
	now := time.Now()
	c := Claims{
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Subject:   u.ID,
			ID:        randomToken()[:16],
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.cfg.JWTSecret)
}

// ---- refresh tokens: opaque, stored hashed, rotated on every use ---------------------------------

var (
	errRefreshInvalid = errors.New("invalid refresh token")
	errRefreshReuse   = errors.New("refresh token reuse detected")
)

type refreshRec struct {
	userID    string
	expires   time.Time
	usedAt    time.Time
	successor string // raw token handed out on first rotation, replayed during the grace window
}

type refreshStore struct {
	mu    sync.Mutex
	recs  map[string]*refreshRec
	ttl   time.Duration
	grace time.Duration
}

func newRefreshStore(ttl, grace time.Duration) *refreshStore {
	return &refreshStore{recs: map[string]*refreshRec{}, ttl: ttl, grace: grace}
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func (r *refreshStore) issueLocked(userID string) string {
	now := time.Now()
	for k, v := range r.recs { // lazy sweep of expired records
		if now.After(v.expires) {
			delete(r.recs, k)
		}
	}
	tok := randomToken()
	r.recs[hashToken(tok)] = &refreshRec{userID: userID, expires: now.Add(r.ttl)}
	return tok
}

func (r *refreshStore) issue(userID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.issueLocked(userID)
}

// rotate consumes a refresh token and returns its successor.
//   - first use: mark used, issue a successor
//   - reuse within the grace window (two tabs/apps refreshing at once): replay the same successor
//   - reuse after the grace window: treat as theft, revoke every session of that user
func (r *refreshStore) rotate(token string) (userID, next string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec := r.recs[hashToken(token)]
	now := time.Now()
	if rec == nil || now.After(rec.expires) {
		return "", "", errRefreshInvalid
	}
	if rec.usedAt.IsZero() {
		rec.usedAt = now
		rec.successor = r.issueLocked(rec.userID)
		return rec.userID, rec.successor, nil
	}
	if now.Sub(rec.usedAt) <= r.grace && rec.successor != "" {
		return rec.userID, rec.successor, nil
	}
	r.revokeUserLocked(rec.userID)
	return "", "", errRefreshReuse
}

func (r *refreshStore) revokeUserLocked(userID string) {
	for k, v := range r.recs {
		if v.userID == userID {
			delete(r.recs, k)
		}
	}
}

func (r *refreshStore) revokeByToken(token string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec := r.recs[hashToken(token)]; rec != nil {
		r.revokeUserLocked(rec.userID)
	}
}

// ---- login throttle: 5 failures per username per minute ------------------------------------------

type throttle struct {
	mu   sync.Mutex
	fail map[string][]time.Time
}

func (t *throttle) prune(key string) []time.Time {
	cut := time.Now().Add(-time.Minute)
	var keep []time.Time
	for _, ts := range t.fail[key] {
		if ts.After(cut) {
			keep = append(keep, ts)
		}
	}
	t.fail[key] = keep
	return keep
}
func (t *throttle) allow(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.prune(key)) < 5
}
func (t *throttle) record(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.fail[key] = append(t.prune(key), time.Now())
}
func (t *throttle) reset(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.fail, key)
}

// ---- cookies -----------------------------------------------------------------------------------

func (s *Server) cookie(name, value, path string, maxAge int, httpOnly bool) *http.Cookie {
	return &http.Cookie{Name: name, Value: value, Path: path, MaxAge: maxAge, HttpOnly: httpOnly,
		Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode}
}

func (s *Server) writeSession(w http.ResponseWriter, u *User, refresh, csrf string) error {
	access, err := s.signAccess(u)
	if err != nil {
		return err
	}
	http.SetCookie(w, s.cookie(cookieSession, access, "/", int(s.cfg.AccessTTL.Seconds()), true))
	http.SetCookie(w, s.cookie(cookieRefresh, refresh, "/api/auth", int(s.cfg.RefreshTTL.Seconds()), true))
	http.SetCookie(w, s.cookie(cookieCSRF, csrf, "/", int(s.cfg.RefreshTTL.Seconds()), false))
	return nil
}

func (s *Server) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, s.cookie(cookieSession, "", "/", -1, true))
	http.SetCookie(w, s.cookie(cookieRefresh, "", "/api/auth", -1, true))
	http.SetCookie(w, s.cookie(cookieCSRF, "", "/", -1, false))
}

// ---- handlers ----------------------------------------------------------------------------------

type meView struct {
	User        map[string]string `json:"user"`
	Permissions []string          `json:"permissions"`
	Outlet      map[string]string `json:"outlet"`
}

func (s *Server) me(u *User) meView {
	return meView{
		User:        map[string]string{"id": u.ID, "username": u.Username, "name": u.Name, "role": u.Role},
		Permissions: permsFor(u.Role),
		Outlet:      map[string]string{"name": s.store.GetSettings().StoreName},
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	key := strings.ToLower(strings.TrimSpace(in.Username))
	if !s.throttle.allow(key) {
		writeErr(w, http.StatusTooManyRequests, "too many failed attempts, try again in a minute")
		return
	}
	u := s.usersByName[key]
	hash := s.dummyHash // compare against a dummy hash for unknown users: no user enumeration by timing
	if u != nil {
		hash = u.hash
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte(in.Password)); err != nil || u == nil {
		s.throttle.record(key)
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	s.throttle.reset(key)
	if err := s.writeSession(w, u, s.refresh.issue(u.ID), randomToken()); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not create session")
		return
	}
	writeJSON(w, http.StatusOK, s.me(u))
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(cookieRefresh)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "no refresh token")
		return
	}
	userID, next, err := s.refresh.rotate(c.Value)
	u := s.usersByID[userID]
	if err != nil || u == nil {
		s.clearSession(w)
		writeErr(w, http.StatusUnauthorized, "session expired, please sign in again")
		return
	}
	csrf := randomToken()
	if cc, err := r.Cookie(cookieCSRF); err == nil && cc.Value != "" {
		csrf = cc.Value // keep it stable so concurrent tabs stay valid
	}
	if err := s.writeSession(w, u, next, csrf); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not create session")
		return
	}
	writeJSON(w, http.StatusOK, s.me(u))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieRefresh); err == nil {
		s.refresh.revokeByToken(c.Value) // server-side invalidation, not just cookie removal
	}
	s.clearSession(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.me(userFrom(r)))
}
