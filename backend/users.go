package main

import (
	_ "embed"
	"encoding/json"

	"golang.org/x/crypto/bcrypt"
)

//go:embed seed/users.json
var seedUsersJSON []byte

type seedUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

type User struct {
	ID, Username, Name, Role string
	hash                     []byte
}

// Permissions are resolved from the role on every request (never trusted from the token),
// so changing a role takes effect immediately.
var rolePerms = map[string][]string{
	"cashier": {"products.read", "cart.use", "orders.create", "orders.read", "settings.read"},
	"manager": {"products.read", "cart.use", "orders.create", "orders.read", "settings.read", "settings.write"},
}

func permsFor(role string) []string {
	p := rolePerms[role]
	out := make([]string, len(p))
	copy(out, p)
	return out
}

func hasPerm(role, perm string) bool {
	for _, p := range rolePerms[role] {
		if p == perm {
			return true
		}
	}
	return false
}

func loadUsers() (byName, byID map[string]*User, err error) {
	var seeds []seedUser
	if err = json.Unmarshal(seedUsersJSON, &seeds); err != nil {
		return nil, nil, err
	}
	byName, byID = map[string]*User{}, map[string]*User{}
	for _, s := range seeds {
		h, err := bcrypt.GenerateFromPassword([]byte(s.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, err
		}
		u := &User{ID: s.ID, Username: s.Username, Name: s.Name, Role: s.Role, hash: h}
		byName[u.Username], byID[u.ID] = u, u
	}
	return byName, byID, nil
}
