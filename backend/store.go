package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	errNotFound   = errors.New("not found")
	errStock      = errors.New("not enough stock")
	errBadRequest = errors.New("bad request")
	errEmptyCart  = errors.New("cart is empty")
)

type Product struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Price    int64  `json:"price"`
	Stock    int    `json:"stock"`
}

type Plan struct {
	ID   string
	Name string
	Bps  int64 // basis points of the unit price
}

var plans = []Plan{
	{"basic-1y", "Basic 1 Year", 500},
	{"plus-2y", "Plus 2 Years", 900},
	{"max-3y", "Max 3 Years (incl. on-site service)", 1200},
}

func findPlan(id string) *Plan {
	for i := range plans {
		if plans[i].ID == id {
			return &plans[i]
		}
	}
	return nil
}

// Server is the single source of truth for plan pricing; clients only display it.
func planPrice(unit int64, p Plan) int64 { return (unit*p.Bps/10000 + 500) / 1000 * 1000 }

type PlanView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int64  `json:"price"` // per unit
}

type CartItemView struct {
	SKU         string     `json:"sku"`
	Name        string     `json:"name"`
	Price       int64      `json:"price"`
	Qty         int        `json:"qty"`
	Plan        *PlanView  `json:"plan"`
	PlanOptions []PlanView `json:"planOptions"`
	LineTotal   int64      `json:"lineTotal"`
}

type CartView struct {
	Items []CartItemView `json:"items"`
	Count int            `json:"count"`
	Total int64          `json:"total"`
}

type cartLine struct {
	SKU    string
	Qty    int
	PlanID string
}

type Order struct {
	ID            string         `json:"id"`
	CashierID     string         `json:"cashierId"`
	CashierName   string         `json:"cashierName"`
	Items         []CartItemView `json:"items"`
	Total         int64          `json:"total"`
	PaymentMethod string         `json:"paymentMethod"`
	ReceiptFooter string         `json:"receiptFooter"`
	CreatedAt     time.Time      `json:"createdAt"`
}

type Settings struct {
	StoreName     string `json:"storeName"`
	ReceiptFooter string `json:"receiptFooter"`
}

type Dashboard struct {
	TodaySales  int64 `json:"todaySales"`
	OrdersToday int   `json:"ordersToday"`
	PlansSold   int   `json:"plansSold"`
	LowStock    int   `json:"lowStock"`
}

type Store struct {
	mu       sync.Mutex
	products map[string]*Product
	carts    map[string][]cartLine // by user id
	orders   []Order
	settings Settings
	seq      int
}

func NewStore() *Store {
	s := &Store{
		products: map[string]*Product{},
		carts:    map[string][]cartLine{},
		settings: Settings{StoreName: "ApplianceHub Jakarta", ReceiptFooter: "Thank you for shopping with us!"},
	}
	for _, p := range []Product{
		{"FRG-410", "Inverter Refrigerator 410L", "Refrigerator", 6490000, 12},
		{"WSH-900", "Front Load Washer 9kg", "Washing Machine", 5290000, 8},
		{"AC-1PK", "Split AC 1 PK Inverter", "Air Conditioner", 4390000, 20},
		{"TV-55U", "Smart TV 55\" 4K", "Television", 7990000, 5},
		{"MWV-25", "Microwave Oven 25L", "Kitchen", 1290000, 30},
		{"RCK-18", "Digital Rice Cooker 1.8L", "Kitchen", 890000, 40},
		{"VAC-RB", "Robot Vacuum Cleaner", "Home Care", 3490000, 6},
		{"WTR-HT", "Water Heater 15L", "Home Care", 1790000, 14},
	} {
		p := p
		s.products[p.SKU] = &p
	}
	// two historical orders so the dashboard is not empty on first run
	s.seedOrder("u-1", "Dewi Cashier", "card", []cartLine{{"WSH-900", 1, "plus-2y"}, {"MWV-25", 2, ""}})
	s.seedOrder("u-1", "Dewi Cashier", "cash", []cartLine{{"FRG-410", 1, ""}})
	return s
}

func (s *Store) seedOrder(uid, name, method string, lines []cartLine) {
	view := s.viewLocked(lines)
	s.seq++
	s.orders = append(s.orders, Order{ID: fmt.Sprintf("ORD-%04d", s.seq), CashierID: uid, CashierName: name,
		Items: view.Items, Total: view.Total, PaymentMethod: method, ReceiptFooter: s.settings.ReceiptFooter, CreatedAt: time.Now()})
}

func (s *Store) ListProducts() []Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Product, 0, len(s.products))
	for _, p := range s.products {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SKU < out[j].SKU })
	return out
}

func (s *Store) GetSettings() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

func (s *Store) UpdateSettings(in Settings) (Settings, error) {
	in.StoreName, in.ReceiptFooter = strings.TrimSpace(in.StoreName), strings.TrimSpace(in.ReceiptFooter)
	if in.StoreName == "" || len(in.StoreName) > 80 || len(in.ReceiptFooter) > 160 {
		return Settings{}, errBadRequest
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = in
	return s.settings, nil
}

// ---- cart ------------------------------------------------------------------------------------

func (s *Store) viewLocked(lines []cartLine) CartView {
	v := CartView{Items: []CartItemView{}}
	for _, l := range lines {
		p := s.products[l.SKU]
		if p == nil {
			continue
		}
		it := CartItemView{SKU: p.SKU, Name: p.Name, Price: p.Price, Qty: l.Qty, PlanOptions: []PlanView{}}
		for _, pl := range plans {
			opt := PlanView{ID: pl.ID, Name: pl.Name, Price: planPrice(p.Price, pl)}
			it.PlanOptions = append(it.PlanOptions, opt)
			if pl.ID == l.PlanID {
				o := opt
				it.Plan = &o
			}
		}
		unit := p.Price
		if it.Plan != nil {
			unit += it.Plan.Price
		}
		it.LineTotal = unit * int64(l.Qty)
		v.Items = append(v.Items, it)
		v.Count += l.Qty
		v.Total += it.LineTotal
	}
	return v
}

func (s *Store) Cart(uid string) CartView {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.viewLocked(s.carts[uid])
}

func (s *Store) lineIdx(uid, sku string) int {
	for i, l := range s.carts[uid] {
		if l.SKU == sku {
			return i
		}
	}
	return -1
}

func (s *Store) CartAdd(uid, sku string) (CartView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.products[sku]
	if p == nil {
		return CartView{}, errNotFound
	}
	if i := s.lineIdx(uid, sku); i >= 0 {
		if s.carts[uid][i].Qty+1 > p.Stock {
			return CartView{}, errStock
		}
		s.carts[uid][i].Qty++
	} else {
		if p.Stock < 1 {
			return CartView{}, errStock
		}
		s.carts[uid] = append(s.carts[uid], cartLine{SKU: sku, Qty: 1})
	}
	return s.viewLocked(s.carts[uid]), nil
}

func (s *Store) CartSetQty(uid, sku string, qty int) (CartView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.lineIdx(uid, sku)
	if i < 0 {
		return CartView{}, errNotFound
	}
	switch {
	case qty <= 0:
		s.carts[uid] = append(s.carts[uid][:i], s.carts[uid][i+1:]...)
	case qty > s.products[sku].Stock:
		return CartView{}, errStock
	default:
		s.carts[uid][i].Qty = qty
	}
	return s.viewLocked(s.carts[uid]), nil
}

func (s *Store) CartSetPlan(uid, sku, planID string) (CartView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.lineIdx(uid, sku)
	if i < 0 {
		return CartView{}, errNotFound
	}
	if planID != "" && findPlan(planID) == nil {
		return CartView{}, errBadRequest
	}
	s.carts[uid][i].PlanID = planID
	return s.viewLocked(s.carts[uid]), nil
}

func (s *Store) CartClear(uid string) CartView {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.carts, uid)
	return s.viewLocked(nil)
}

// ---- orders ----------------------------------------------------------------------------------

var paymentMethods = map[string]bool{"cash": true, "card": true, "qris": true}

func (s *Store) Checkout(uid, cashier, method string) (Order, error) {
	if !paymentMethods[method] {
		return Order{}, errBadRequest
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lines := s.carts[uid]
	if len(lines) == 0 {
		return Order{}, errEmptyCart
	}
	for _, l := range lines { // re-validate stock at the moment of payment
		if p := s.products[l.SKU]; p == nil || l.Qty > p.Stock {
			return Order{}, errStock
		}
	}
	view := s.viewLocked(lines)
	for _, l := range lines {
		s.products[l.SKU].Stock -= l.Qty
	}
	s.seq++
	o := Order{ID: fmt.Sprintf("ORD-%04d", s.seq), CashierID: uid, CashierName: cashier, Items: view.Items,
		Total: view.Total, PaymentMethod: method, ReceiptFooter: s.settings.ReceiptFooter, CreatedAt: time.Now()}
	s.orders = append(s.orders, o)
	delete(s.carts, uid)
	return o, nil
}

func (s *Store) Orders() []Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Order, len(s.orders))
	for i, o := range s.orders {
		out[len(s.orders)-1-i] = o // newest first
	}
	return out
}

func (s *Store) Dashboard() Dashboard {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := Dashboard{}
	y, m, day := time.Now().Date()
	for _, o := range s.orders {
		if oy, om, od := o.CreatedAt.Date(); oy == y && om == m && od == day {
			d.TodaySales += o.Total
			d.OrdersToday++
			for _, it := range o.Items {
				if it.Plan != nil {
					d.PlansSold += it.Qty
				}
			}
		}
	}
	for _, p := range s.products {
		if p.Stock < 10 {
			d.LowStock++
		}
	}
	return d
}
