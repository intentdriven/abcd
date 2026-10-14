// Package githubpagestest is an in-process fake of the slice of the GitHub
// REST API the github-pages hosting adapter calls, for tests only. The adapter
// speaks through the forge the way the site setup's forge stage does — `gh api`
// as the person who invoked the verb, never a token abcd holds — so the fake
// answers the adapter's API calls directly rather than over a socket, and
// fails them the way `gh api` does: an error naming the HTTP status.
//
// It answers exactly these routes, and anything else is a 404:
//
//	GET  repos/{owner}/{repo}         the repository: private, has_pages
//	GET  repos/{owner}/{repo}/pages   the Pages site: build_type, cname, html_url
//	POST repos/{owner}/{repo}/pages   create the Pages site
//	PUT  repos/{owner}/{repo}/pages   change its source or custom domain
package githubpagestest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Route keys name a route for Fail and for the call log.
const (
	GetRepo     = "GET repo"
	GetPages    = "GET pages"
	CreatePages = "POST pages"
	UpdatePages = "PUT pages"
)

// Pages is one repository's Pages site.
type Pages struct {
	BuildType string
	CNAME     string
}

// Repo is one repository the fake holds.
type Repo struct {
	Private bool
	// Pages is nil while the repository has no Pages site.
	Pages *Pages
}

// Server is the fake. Its fields are the forge's state; lock Mu to read them
// while a call may be in flight.
type Server struct {
	Mu    sync.Mutex
	Repos map[string]*Repo
	// Fail makes a route answer with this error.
	Fail map[string]error
	// Calls logs each call as "<route key> <body>".
	Calls []string
}

// New is a fake holding one public repository, acme/widgets, with no Pages
// site yet.
func New() *Server {
	return &Server{Repos: map[string]*Repo{"acme/widgets": {}}, Fail: map[string]error{}}
}

// CallLog is a copy of the calls made so far.
func (s *Server) CallLog() []string {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return append([]string(nil), s.Calls...)
}

// status is the error `gh api` returns for an HTTP status.
func status(code int, text string) error {
	return fmt.Errorf("exit status 1 (gh: %s (HTTP %d))", text, code)
}

// API answers one call, shaped as the adapter's API.
func (s *Server) API(_ context.Context, method, path string, body []byte) ([]byte, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	rest, ok := strings.CutPrefix(path, "repos/")
	if !ok {
		return nil, status(404, "Not Found")
	}
	parts := strings.Split(rest, "/")
	key := ""
	switch {
	case len(parts) == 2 && method == "GET":
		key = GetRepo
	case len(parts) == 3 && parts[2] == "pages":
		key = method + " pages"
	default:
		return nil, status(404, "Not Found")
	}
	s.Calls = append(s.Calls, strings.TrimSpace(key+" "+string(body)))
	if err := s.Fail[key]; err != nil {
		return nil, err
	}
	full := parts[0] + "/" + parts[1]
	repo, ok := s.Repos[full]
	if !ok {
		return nil, status(404, "Not Found")
	}
	switch key {
	case GetRepo:
		return json.Marshal(map[string]any{"full_name": full, "private": repo.Private, "has_pages": repo.Pages != nil})
	case GetPages:
		if repo.Pages == nil {
			return nil, status(404, "Not Found")
		}
		addr := "https://" + strings.ToLower(parts[0]) + ".github.io/" + parts[1] + "/"
		if repo.Pages.CNAME != "" {
			addr = "https://" + repo.Pages.CNAME + "/"
		}
		return json.Marshal(map[string]any{"build_type": repo.Pages.BuildType, "cname": nilIfEmpty(repo.Pages.CNAME), "html_url": addr})
	case CreatePages:
		if repo.Pages != nil {
			return nil, status(409, "GitHub Pages is already enabled.")
		}
		var req struct {
			BuildType string `json:"build_type"`
		}
		if json.Unmarshal(body, &req) != nil || req.BuildType == "" {
			return nil, status(422, "Invalid request.")
		}
		repo.Pages = &Pages{BuildType: req.BuildType}
		return []byte("{}"), nil
	case UpdatePages:
		if repo.Pages == nil {
			return nil, status(404, "Not Found")
		}
		var req map[string]*string
		if json.Unmarshal(body, &req) != nil {
			return nil, status(422, "Invalid request.")
		}
		if v, ok := req["cname"]; ok && v != nil {
			repo.Pages.CNAME = *v
		}
		if v, ok := req["build_type"]; ok && v != nil {
			repo.Pages.BuildType = *v
		}
		return nil, nil
	}
	return nil, status(404, "Not Found")
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
