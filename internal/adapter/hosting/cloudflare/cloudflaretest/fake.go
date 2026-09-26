// Package cloudflaretest is an in-process fake of the slice of the Cloudflare
// v4 API the cloudflare hosting adapter calls, for tests only. No test in this
// repository reaches the real service: the adapter and the verb above it are
// exercised against this server, and every route it answers can be made to
// fail, so each error path is a test rather than a hope.
//
// The fake answers the documented shapes (the v4 envelope: success, errors,
// result, result_info) for exactly these routes:
//
//	GET  /accounts
//	GET  /accounts/{account}/workers/workers
//	POST /accounts/{account}/workers/workers
//	GET  /accounts/{account}/workers/domains
//	PUT  /accounts/{account}/workers/domains
//	GET  /accounts/{account}/workers/subdomain
//
// Anything else is a 404, so a request the adapter was not written to make
// fails the test that provoked it.
package cloudflaretest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Token is the credential the fake accepts. Any other bearer is a 403.
const Token = "cf-fake-token-0000000000000000000000000000"

// Account is the one account the default fake holds.
const Account = "0123456789abcdef0123456789abcdef"

// Subdomain is the account's workers.dev subdomain.
const Subdomain = "example-sub"

// Route keys name a route for Fail and for the call log.
const (
	ListAccounts  = "GET accounts"
	ListWorkers   = "GET workers"
	CreateWorker  = "POST workers"
	ListDomains   = "GET domains"
	AttachDomain  = "PUT domains"
	GetSubdomain  = "GET subdomain"
	unknownRoute  = "unknown"
	envelopeError = 10000
)

// Server is the fake. Its fields are the host's state; lock with Mu to read
// them while a request may be in flight.
type Server struct {
	*httptest.Server
	Mu sync.Mutex
	// Accounts the token reaches.
	Accounts []string
	// Workers by name.
	Workers map[string]bool
	// Domains maps a hostname to the Worker it is routed to.
	Domains map[string]string
	// Fail answers a route with this HTTP status instead of serving it.
	Fail map[string]int
	// PageSize splits the Workers list into pages of this size (0 = one page).
	PageSize int
	// Calls is every request received, as its route key.
	Calls []string
	// Bodies holds each write's decoded JSON body, keyed by route.
	Bodies map[string][]map[string]any
	// Auth records every Authorization header value received, so a test can
	// assert the credential went where it should and nowhere else.
	Auth []string
}

// New starts a fake with one account, no Workers and no domains.
func New(t testing.TB) *Server {
	t.Helper()
	s := &Server{
		Accounts: []string{Account},
		Workers:  map[string]bool{},
		Domains:  map[string]string{},
		Fail:     map[string]int{},
		Bodies:   map[string][]map[string]any{},
	}
	s.Server = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.Close)
	return s
}

// Writes is how many write requests the fake received.
func (s *Server) Writes() int {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	n := 0
	for _, c := range s.Calls {
		if !strings.HasPrefix(c, "GET ") {
			n++
		}
	}
	return n
}

// CallLog returns a copy of the call log.
func (s *Server) CallLog() []string {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return append([]string(nil), s.Calls...)
}

func (s *Server) route(r *http.Request) (key, account string) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	switch {
	case len(parts) == 1 && parts[0] == "accounts" && r.Method == http.MethodGet:
		return ListAccounts, ""
	case len(parts) == 4 && parts[0] == "accounts" && parts[2] == "workers":
		account = parts[1]
		switch {
		case parts[3] == "workers" && r.Method == http.MethodGet:
			return ListWorkers, account
		case parts[3] == "workers" && r.Method == http.MethodPost:
			return CreateWorker, account
		case parts[3] == "domains" && r.Method == http.MethodGet:
			return ListDomains, account
		case parts[3] == "domains" && r.Method == http.MethodPut:
			return AttachDomain, account
		case parts[3] == "subdomain" && r.Method == http.MethodGet:
			return GetSubdomain, account
		}
	}
	return unknownRoute, ""
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	key, account := s.route(r)
	s.Calls = append(s.Calls, key)
	s.Auth = append(s.Auth, r.Header.Get("Authorization"))
	if r.Header.Get("Authorization") != "Bearer "+Token {
		fail(w, http.StatusForbidden, "Authentication error")
		return
	}
	if code, ok := s.Fail[key]; ok {
		fail(w, code, "injected failure on "+key)
		return
	}
	if key == unknownRoute {
		fail(w, http.StatusNotFound, "no such route")
		return
	}
	if account != "" && !contains(s.Accounts, account) {
		fail(w, http.StatusForbidden, "account not reachable")
		return
	}
	var body map[string]any
	if r.Method != http.MethodGet {
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err := json.Unmarshal(raw, &body); err != nil {
			fail(w, http.StatusBadRequest, "body is not JSON")
			return
		}
		s.Bodies[key] = append(s.Bodies[key], body)
	}
	switch key {
	case ListAccounts:
		var out []map[string]string
		for _, a := range s.Accounts {
			out = append(out, map[string]string{"id": a, "name": "account " + a[:4]})
		}
		ok(w, out, nil)
	case ListWorkers:
		names := make([]string, 0, len(s.Workers))
		for n := range s.Workers {
			names = append(names, n)
		}
		sort.Strings(names)
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		size := s.PageSize
		if size <= 0 {
			size = len(names) + 1
		}
		total := (len(names) + size - 1) / size
		if total == 0 {
			total = 1
		}
		lo, hi := (page-1)*size, page*size
		if lo > len(names) {
			lo = len(names)
		}
		if hi > len(names) {
			hi = len(names)
		}
		var out []map[string]any
		for _, n := range names[lo:hi] {
			out = append(out, map[string]any{"id": "id-" + n, "name": n})
		}
		ok(w, out, map[string]int{"page": page, "total_pages": total})
	case CreateWorker:
		name, _ := body["name"].(string)
		if name == "" || s.Workers[name] {
			fail(w, http.StatusConflict, "worker exists or has no name")
			return
		}
		s.Workers[name] = true
		ok(w, map[string]any{"id": "id-" + name, "name": name}, nil)
	case ListDomains:
		host := r.URL.Query().Get("hostname")
		var out []map[string]string
		for h, svc := range s.Domains {
			if host == "" || h == host {
				out = append(out, map[string]string{"hostname": h, "service": svc})
			}
		}
		ok(w, out, nil)
	case AttachDomain:
		host, _ := body["hostname"].(string)
		svc, _ := body["service"].(string)
		if !s.Workers[svc] {
			fail(w, http.StatusBadRequest, "no such worker")
			return
		}
		s.Domains[host] = svc
		ok(w, map[string]string{"hostname": host, "service": svc}, nil)
	case GetSubdomain:
		ok(w, map[string]string{"subdomain": Subdomain}, nil)
	}
}

func ok(w http.ResponseWriter, result any, info any) {
	env := map[string]any{"success": true, "errors": []any{}, "messages": []any{}, "result": result}
	if info != nil {
		env["result_info"] = info
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(env)
}

func fail(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false, "result": nil,
		"errors": []map[string]any{{"code": envelopeError, "message": msg}},
	})
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
