package session

import (
	"net/http"
	"regexp"
	"strings"
)

// A local session's relay serves the session page itself, so nothing goes to
// abcdesign.app (adr-2610161400542916, and adr-2610032150581128 as amended
// 2026-10-16). It serves the page's embedded files from a fixed route table,
// never from a requested path: the four assets at the site's root, and the
// HTML, carrying web.RelayMarker, at /<owner>/<repo>/brief/ for a repository
// the relay serves.

// pageAssets are the page's files the relay serves, by path, with their
// content types.
var pageAssets = map[string]string{
	"/session.js":          "text/javascript; charset=utf-8",
	"/session.css":         "text/css; charset=utf-8",
	"/vendor/collab.js":    "text/javascript; charset=utf-8",
	"/vendor/LICENSES.txt": "text/plain; charset=utf-8",
}

// briefPathRe is /<owner>/<repo>/brief/, with or without its slash: the
// page adds it.
var briefPathRe = regexp.MustCompile(`^/([A-Za-z0-9][A-Za-z0-9-]{0,38})/([A-Za-z0-9._-]{1,100})/brief/?$`)

// servePage answers a page path, reporting whether path was one. Only GET
// and HEAD are taken.
func (s *Server) servePage(w http.ResponseWriter, r *http.Request) bool {
	if !s.cfg.ServePage {
		return false
	}
	path := r.URL.Path
	h := w.Header()
	if kind, ok := pageAssets[path]; ok {
		if !pageMethod(w, r) {
			return true
		}
		h.Set("Content-Type", kind)
		_, _ = w.Write(s.page[strings.TrimPrefix(path, "/")])
		return true
	}
	if path == "/" && len(s.cfg.Repos) == 1 {
		if !pageMethod(w, r) {
			return true
		}
		http.Redirect(w, r, "/"+s.cfg.Repos[0]+"/brief/", http.StatusFound)
		return true
	}
	m := briefPathRe.FindStringSubmatch(path)
	if m == nil || m[2] == "." || m[2] == ".." || !s.relay.Serves(m[1]+"/"+m[2]) {
		return false
	}
	if !pageMethod(w, r) {
		return true
	}
	// The page refuses to run framed; a relay can also say so in headers,
	// which a static host cannot.
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Content-Security-Policy", "frame-ancestors 'none'")
	_, _ = w.Write(s.page["index.html"])
	return true
}

func pageMethod(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// hostAllowed judges a request's Host against the names the relay answers:
// any, when none were given; otherwise exactly one of them, compared without
// case.
func (s *Server) hostAllowed(host string) bool {
	if s.hosts == nil {
		return true
	}
	return s.hosts[strings.ToLower(host)]
}
