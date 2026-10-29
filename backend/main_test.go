package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// ---- helpers ---------------------------------------------------------------------------------

type client struct {
	t    *testing.T
	base string
	http *http.Client
	jar  *cookiejar.Jar
}

func newTestServer(t *testing.T, mutate func(*Config)) (*httptest.Server, *Server) {
	t.Helper()
	cfg := Config{Port: "0", JWTSecret: []byte("test-secret-test-secret-test-secret-123"),
		AccessTTL: 15 * time.Minute, RefreshTTL: time.Hour, RefreshGrace: 30 * time.Second}
	if mutate != nil {
		mutate(&cfg)
	}
	s, err := NewServer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Routes())
	t.Cleanup(ts.Close)
	return ts, s
}

func newClient(t *testing.T, ts *httptest.Server) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, base: ts.URL, jar: jar, http: &http.Client{Jar: jar}}
}

func (c *client) csrf() string {
	u, _ := url.Parse(c.base)
	for _, ck := range c.jar.Cookies(u) {
		if ck.Name == cookieCSRF {
			return ck.Value
		}
	}
	return ""
}

func (c *client) do(method, path string, body any, withCSRF bool) (int, map[string]any, []byte) {
	c.t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, c.base+path, rdr)
	req.Header.Set("Content-Type", "application/json")
	if withCSRF {
		req.Header.Set("X-CSRF-Token", c.csrf())
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(res.Body)
	var m map[string]any
	_ = json.Unmarshal(buf.Bytes(), &m)
	return res.StatusCode, m, buf.Bytes()
}

func seedCreds(t *testing.T, username string) map[string]string {
	var seeds []seedUser
	if err := json.Unmarshal(seedUsersJSON, &seeds); err != nil {
		t.Fatal(err)
	}
	for _, s := range seeds {
		if s.Username == username {
			return map[string]string{"username": s.Username, "password": s.Password}
		}
	}
	t.Fatalf("seed user %s not found", username)
	return nil
}

func login(t *testing.T, c *client, username string) {
	t.Helper()
	if code, _, raw := c.do("POST", "/api/auth/login", seedCreds(t, username), false); code != 200 {
		t.Fatalf("login %s: %d %s", username, code, raw)
	}
}

// ---- tests -----------------------------------------------------------------------------------

func TestLoginSetsHardenedCookiesAndMe(t *testing.T) {
	ts, _ := newTestServer(t, nil)
	c := newClient(t, ts)

	if code, _, _ := c.do("POST", "/api/auth/login", map[string]string{"username": "cashier", "password": "wrong"}, false); code != 401 {
		t.Fatalf("bad password: want 401 got %d", code)
	}
	if code, _, _ := c.do("POST", "/api/auth/login", map[string]string{"username": "nobody", "password": "x"}, false); code != 401 {
		t.Fatalf("unknown user: want 401 got %d", code)
	}

	body, _ := json.Marshal(seedCreds(t, "cashier"))
	res, err := http.Post(ts.URL+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("login failed: %v %v", err, res)
	}
	got := map[string]*http.Cookie{}
	for _, ck := range res.Cookies() {
		got[ck.Name] = ck
	}
	if s := got[cookieSession]; s == nil || !s.HttpOnly || s.Path != "/" || s.SameSite != http.SameSiteLaxMode {
		t.Errorf("session cookie not hardened: %+v", s)
	}
	if r := got[cookieRefresh]; r == nil || !r.HttpOnly || r.Path != "/api/auth" {
		t.Errorf("refresh cookie must be HttpOnly and scoped to /api/auth: %+v", r)
	}
	if x := got[cookieCSRF]; x == nil || x.HttpOnly {
		t.Errorf("csrf cookie must be readable by JS: %+v", x)
	}

	login(t, c, "cashier")
	code, me, _ := c.do("GET", "/api/me", nil, false)
	if code != 200 || me["user"].(map[string]any)["role"] != "cashier" {
		t.Fatalf("me: %d %v", code, me)
	}
}

func TestUnauthenticatedAndBadTokens(t *testing.T) {
	ts, s := newTestServer(t, nil)
	c := newClient(t, ts)
	for _, p := range []string{"/api/me", "/api/products", "/api/cart", "/api/orders", "/api/dashboard", "/api/settings"} {
		if code, _, _ := c.do("GET", p, nil, false); code != 401 {
			t.Errorf("%s without auth: want 401 got %d", p, code)
		}
	}
	req, _ := http.NewRequest("GET", ts.URL+"/api/me", nil)
	req.Header.Set("Authorization", "Bearer not.a.jwt")
	if res, _ := http.DefaultClient.Do(req); res.StatusCode != 401 {
		t.Errorf("garbage bearer: want 401 got %d", res.StatusCode)
	}
	// token signed with another secret must be rejected
	other := &Server{cfg: Config{JWTSecret: []byte("a-completely-different-secret-value-!!"), AccessTTL: time.Minute}}
	forged, _ := other.signAccess(s.usersByName["manager"])
	req, _ = http.NewRequest("GET", ts.URL+"/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+forged)
	if res, _ := http.DefaultClient.Do(req); res.StatusCode != 401 {
		t.Errorf("forged token: want 401 got %d", res.StatusCode)
	}
}

func TestBearerHeaderWorksAndSkipsCSRF(t *testing.T) {
	ts, s := newTestServer(t, nil)
	tok, _ := s.signAccess(s.usersByName["cashier"])
	req, _ := http.NewRequest("POST", ts.URL+"/api/cart/items", bytes.NewReader([]byte(`{"sku":"FRG-410"}`)))
	req.Header.Set("Authorization", "Bearer "+tok)
	if res, _ := http.DefaultClient.Do(req); res.StatusCode != 200 {
		t.Errorf("bearer mutation: want 200 got %d", res.StatusCode)
	}
}

func TestCSRFRequiredForCookieAuth(t *testing.T) {
	ts, _ := newTestServer(t, nil)
	c := newClient(t, ts)
	login(t, c, "cashier")
	if code, _, _ := c.do("POST", "/api/cart/items", map[string]string{"sku": "FRG-410"}, false); code != 403 {
		t.Errorf("mutation without CSRF header: want 403 got %d", code)
	}
	if code, _, _ := c.do("POST", "/api/cart/items", map[string]string{"sku": "FRG-410"}, true); code != 200 {
		t.Errorf("mutation with CSRF header: want 200 got %d", code)
	}
}

func TestRolePermissions(t *testing.T) {
	ts, _ := newTestServer(t, nil)
	cashier := newClient(t, ts)
	login(t, cashier, "cashier")
	update := map[string]string{"storeName": "Changed Store", "receiptFooter": "bye"}
	if code, _, _ := cashier.do("GET", "/api/settings", nil, false); code != 200 {
		t.Errorf("cashier read settings: got %d", code)
	}
	if code, _, _ := cashier.do("PUT", "/api/settings", update, true); code != 403 {
		t.Errorf("cashier write settings: want 403 got %d", code)
	}
	manager := newClient(t, ts)
	login(t, manager, "manager")
	if code, _, _ := manager.do("PUT", "/api/settings", update, true); code != 200 {
		t.Errorf("manager write settings: want 200 got %d", code)
	}
	if code, _, _ := manager.do("PUT", "/api/settings", map[string]string{"storeName": "  ", "receiptFooter": ""}, true); code != 400 {
		t.Errorf("empty store name: want 400 got %d", code)
	}
	_, me, _ := cashier.do("GET", "/api/me", nil, false)
	if me["outlet"].(map[string]any)["name"] != "Changed Store" {
		t.Errorf("outlet name should follow settings, got %v", me["outlet"])
	}
}

func TestCartPlanPricingAndCheckout(t *testing.T) {
	ts, _ := newTestServer(t, nil)
	c := newClient(t, ts)
	login(t, c, "cashier")

	c.do("POST", "/api/cart/items", map[string]string{"sku": "FRG-410"}, true)
	c.do("POST", "/api/cart/items", map[string]string{"sku": "FRG-410"}, true)
	c.do("POST", "/api/cart/items", map[string]string{"sku": "AC-1PK"}, true)
	code, cart, _ := c.do("PUT", "/api/cart/items/FRG-410/plan", map[string]string{"planId": "plus-2y"}, true)
	if code != 200 {
		t.Fatalf("set plan: %d", code)
	}
	// 2 x (6,490,000 + 584,000) + 4,390,000 = 18,538,000 ; plan price is computed by the server
	if cart["total"].(float64) != 18538000 || cart["count"].(float64) != 3 {
		t.Fatalf("cart total/count wrong: %v", cart)
	}
	if code, _, _ := c.do("PUT", "/api/cart/items/FRG-410/plan", map[string]string{"planId": "bogus"}, true); code != 400 {
		t.Errorf("unknown plan: want 400 got %d", code)
	}
	if code, _, _ := c.do("PATCH", "/api/cart/items/AC-1PK", map[string]int{"qty": 999}, true); code != 409 {
		t.Errorf("qty over stock: want 409 got %d", code)
	}
	if code, _, _ := c.do("POST", "/api/cart/items", map[string]string{"sku": "NOPE"}, true); code != 404 {
		t.Errorf("unknown sku: want 404 got %d", code)
	}
	if code, _, _ := c.do("POST", "/api/orders", map[string]string{"paymentMethod": "bitcoin"}, true); code != 400 {
		t.Errorf("bad payment method: want 400 got %d", code)
	}

	code, order, _ := c.do("POST", "/api/orders", map[string]string{"paymentMethod": "card"}, true)
	if code != 201 || order["total"].(float64) != 18538000 {
		t.Fatalf("checkout: %d %v", code, order)
	}
	_, cart, _ = c.do("GET", "/api/cart", nil, false)
	if cart["count"].(float64) != 0 {
		t.Errorf("cart should be empty after checkout: %v", cart)
	}
	if code, _, _ := c.do("POST", "/api/orders", map[string]string{"paymentMethod": "card"}, true); code != 409 {
		t.Errorf("checkout of empty cart: want 409 got %d", code)
	}
	_, _, raw := c.do("GET", "/api/products", nil, false)
	var products []Product
	_ = json.Unmarshal(raw, &products)
	for _, p := range products {
		if p.SKU == "FRG-410" && p.Stock != 10 {
			t.Errorf("stock should drop 12 -> 10, got %d", p.Stock)
		}
	}
	_, _, raw = c.do("GET", "/api/orders", nil, false)
	var orders []Order
	_ = json.Unmarshal(raw, &orders)
	if len(orders) != 3 || orders[0].ID != order["id"] {
		t.Errorf("orders newest first expected, got %d orders", len(orders))
	}
	_, dash, _ := c.do("GET", "/api/dashboard", nil, false)
	if dash["ordersToday"].(float64) != 3 || dash["plansSold"].(float64) != 3 {
		t.Errorf("dashboard wrong: %v", dash)
	}
}

func TestCartsAreIsolatedPerUser(t *testing.T) {
	ts, _ := newTestServer(t, nil)
	a, b := newClient(t, ts), newClient(t, ts)
	login(t, a, "cashier")
	login(t, b, "manager")
	a.do("POST", "/api/cart/items", map[string]string{"sku": "TV-55U"}, true)
	_, cart, _ := b.do("GET", "/api/cart", nil, false)
	if cart["count"].(float64) != 0 {
		t.Errorf("manager must not see the cashier's cart: %v", cart)
	}
}

func TestRefreshRotationGraceAndReuseDetection(t *testing.T) {
	ts, _ := newTestServer(t, func(c *Config) { c.RefreshGrace = 150 * time.Millisecond })
	c := newClient(t, ts)
	login(t, c, "cashier")
	u, _ := url.Parse(ts.URL + "/api/auth")
	oldRefresh := ""
	for _, ck := range c.jar.Cookies(u) {
		if ck.Name == cookieRefresh {
			oldRefresh = ck.Value
		}
	}
	if code, _, _ := c.do("POST", "/api/auth/refresh", nil, true); code != 200 {
		t.Fatalf("first refresh: %d", code)
	}
	// concurrent client still holding the OLD token inside the grace window must succeed (two tabs/apps)
	c2 := newClient(t, ts)
	c2.jar.SetCookies(u, []*http.Cookie{{Name: cookieRefresh, Value: oldRefresh, Path: "/api/auth"}})
	c2.jar.SetCookies(&url.URL{Scheme: "http", Host: u.Host, Path: "/"}, []*http.Cookie{{Name: cookieCSRF, Value: c.csrf(), Path: "/"}})
	if code, _, _ := c2.do("POST", "/api/auth/refresh", nil, true); code != 200 {
		t.Fatalf("refresh inside grace window should succeed, got %d", code)
	}
	time.Sleep(250 * time.Millisecond)
	// after the grace window the same old token is a replay: reject and revoke the whole family
	c3 := newClient(t, ts)
	c3.jar.SetCookies(u, []*http.Cookie{{Name: cookieRefresh, Value: oldRefresh, Path: "/api/auth"}})
	c3.jar.SetCookies(&url.URL{Scheme: "http", Host: u.Host, Path: "/"}, []*http.Cookie{{Name: cookieCSRF, Value: "x", Path: "/"}})
	if code, _, _ := c3.do("POST", "/api/auth/refresh", nil, true); code != 401 {
		t.Fatalf("replayed token after grace: want 401 got %d", code)
	}
	if code, _, _ := c.do("POST", "/api/auth/refresh", nil, true); code != 401 {
		t.Fatalf("legitimate session should be revoked after reuse detection, got %d", code)
	}
}

func TestAccessTokenExpiresAndRefreshRecovers(t *testing.T) {
	ts, _ := newTestServer(t, func(c *Config) { c.AccessTTL = 1 * time.Second })
	c := newClient(t, ts)
	login(t, c, "cashier")
	if code, _, _ := c.do("GET", "/api/me", nil, false); code != 200 {
		t.Fatalf("fresh token: %d", code)
	}
	time.Sleep(2200 * time.Millisecond)
	// the browser drops the expired cookie (MaxAge); emulate by clearing it, refresh cookie remains
	u, _ := url.Parse(ts.URL)
	c.jar.SetCookies(u, []*http.Cookie{{Name: cookieSession, Value: "", Path: "/", MaxAge: -1}})
	if code, _, _ := c.do("GET", "/api/me", nil, false); code != 401 {
		t.Fatalf("expired: want 401 got %d", code)
	}
	if code, _, _ := c.do("POST", "/api/auth/refresh", nil, true); code != 200 {
		t.Fatalf("refresh: %d", code)
	}
	if code, _, _ := c.do("GET", "/api/me", nil, false); code != 200 {
		t.Fatalf("after refresh: %d", code)
	}
}

func TestExpiredJWTRejectedServerSide(t *testing.T) {
	ts, s := newTestServer(t, func(c *Config) { c.AccessTTL = -time.Minute })
	tok, _ := s.signAccess(s.usersByName["cashier"])
	req, _ := http.NewRequest("GET", ts.URL+"/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	if res, _ := http.DefaultClient.Do(req); res.StatusCode != 401 {
		t.Errorf("expired JWT: want 401 got %d", res.StatusCode)
	}
}

func TestLogoutRevokesRefresh(t *testing.T) {
	ts, _ := newTestServer(t, nil)
	c := newClient(t, ts)
	login(t, c, "cashier")
	u, _ := url.Parse(ts.URL + "/api/auth")
	var saved *http.Cookie
	for _, ck := range c.jar.Cookies(u) {
		if ck.Name == cookieRefresh {
			saved = ck
		}
	}
	csrf := c.csrf()
	if code, _, _ := c.do("POST", "/api/auth/logout", nil, true); code != 204 {
		t.Fatalf("logout: %d", code)
	}
	if code, _, _ := c.do("GET", "/api/me", nil, false); code != 401 {
		t.Errorf("after logout /me: want 401 got %d", code)
	}
	// replaying the saved refresh cookie must fail: logout invalidates server-side
	c2 := newClient(t, ts)
	c2.jar.SetCookies(u, []*http.Cookie{saved})
	c2.jar.SetCookies(&url.URL{Scheme: "http", Host: u.Host, Path: "/"}, []*http.Cookie{{Name: cookieCSRF, Value: csrf, Path: "/"}})
	if code, _, _ := c2.do("POST", "/api/auth/refresh", nil, true); code != 401 {
		t.Errorf("refresh after logout: want 401 got %d", code)
	}
}

func TestLoginThrottle(t *testing.T) {
	ts, _ := newTestServer(t, nil)
	c := newClient(t, ts)
	for i := 0; i < 5; i++ {
		c.do("POST", "/api/auth/login", map[string]string{"username": "manager", "password": "nope"}, false)
	}
	// even the right password is refused while throttled
	if code, _, _ := c.do("POST", "/api/auth/login", seedCreds(t, "manager"), false); code != 429 {
		t.Errorf("throttled login: want 429 got %d", code)
	}
}
