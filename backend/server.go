package main

import (
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	cfg         Config
	store       *Store
	refresh     *refreshStore
	throttle    *throttle
	usersByName map[string]*User
	usersByID   map[string]*User
	dummyHash   []byte
}

func NewServer(cfg Config) (*Server, error) {
	byName, byID, err := loadUsers()
	if err != nil {
		return nil, err
	}
	dummy, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg: cfg, store: NewStore(),
		refresh:     newRefreshStore(cfg.RefreshTTL, cfg.RefreshGrace),
		throttle:    &throttle{fail: map[string][]time.Time{}},
		usersByName: byName, usersByID: byID, dummyHash: dummy,
	}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })

	// auth: refresh/logout rely on cookies, so they are CSRF-protected but do not need a valid access token
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.Handle("POST /api/auth/refresh", s.csrf(http.HandlerFunc(s.handleRefresh)))
	mux.Handle("POST /api/auth/logout", s.csrf(http.HandlerFunc(s.handleLogout)))
	mux.Handle("GET /api/me", s.authn(http.HandlerFunc(s.handleMe)))

	// protected: authenticate -> CSRF -> permission
	guard := func(perm string, h http.HandlerFunc) http.Handler {
		return s.authn(s.csrf(s.requirePerm(perm, h)))
	}
	mux.Handle("GET /api/products", guard("products.read", s.handleProducts))
	mux.Handle("GET /api/dashboard", guard("orders.read", s.handleDashboard))

	mux.Handle("GET /api/cart", guard("cart.use", s.handleCartGet))
	mux.Handle("POST /api/cart/items", guard("cart.use", s.handleCartAdd))
	mux.Handle("PATCH /api/cart/items/{sku}", guard("cart.use", s.handleCartQty))
	mux.Handle("PUT /api/cart/items/{sku}/plan", guard("cart.use", s.handleCartPlan))
	mux.Handle("DELETE /api/cart", guard("cart.use", s.handleCartClear))

	mux.Handle("POST /api/orders", guard("orders.create", s.handleCheckout))
	mux.Handle("GET /api/orders", guard("orders.read", s.handleOrders))

	mux.Handle("GET /api/settings", guard("settings.read", s.handleSettingsGet))
	mux.Handle("PUT /api/settings", guard("settings.write", s.handleSettingsPut))

	return withBasics(mux)
}

func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errStock), errors.Is(err, errEmptyCart):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, errBadRequest):
		writeErr(w, http.StatusBadRequest, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, "internal error")
	}
}

func (s *Server) handleProducts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListProducts())
}
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Dashboard())
}
func (s *Server) handleCartGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Cart(userFrom(r).ID))
}
func (s *Server) handleCartAdd(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SKU string `json:"sku"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, err := s.store.CartAdd(userFrom(r).ID, in.SKU)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func (s *Server) handleCartQty(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Qty int `json:"qty"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, err := s.store.CartSetQty(userFrom(r).ID, r.PathValue("sku"), in.Qty)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func (s *Server) handleCartPlan(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PlanID string `json:"planId"` // "" removes the plan
	}
	if !decode(w, r, &in) {
		return
	}
	v, err := s.store.CartSetPlan(userFrom(r).ID, r.PathValue("sku"), in.PlanID)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func (s *Server) handleCartClear(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.CartClear(userFrom(r).ID))
}
func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PaymentMethod string `json:"paymentMethod"`
	}
	if !decode(w, r, &in) {
		return
	}
	u := userFrom(r)
	o, err := s.store.Checkout(u.ID, u.Name, in.PaymentMethod)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, o)
}
func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Orders())
}
func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.GetSettings())
}
func (s *Server) handleSettingsPut(w http.ResponseWriter, r *http.Request) {
	var in Settings
	if !decode(w, r, &in) {
		return
	}
	out, err := s.store.UpdateSettings(in)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
