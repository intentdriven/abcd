package site

// The base path (itd-2610091010348892): the path a site is served below on its
// host. abcd's own site is served at the root of its domain, so its base path
// is "/" and every page links the shared stylesheet as /site.css. A project's
// GitHub Pages address is owner.github.io/<repo>/, so a site built for it links
// /<repo>/site.css instead, and every other root-absolute URL the build writes
// moves under the same prefix — the shape Hugo's baseURL, Astro's and
// VitePress's base and Docusaurus's baseUrl give the same problem.
//
// Every root-absolute URL the build writes goes through sitePath, and the gate
// resolves what it reads against the same prefix, so a build and its check
// agree on where the site's root is.

import (
	"fmt"
	"regexp"
	"strings"
)

// segmentRe is one plain path segment: the URL's unreserved characters, so a
// base path is never escaped, never interpreted and never leaves an attribute.
var segmentRe = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)

// NormaliseBasePath is the base path s names, with one leading and one
// trailing slash: "" and "/" are the root, "widgets" is "/widgets/". It
// refuses anything but plain path segments — no scheme or host, no empty, "."
// or ".." segment, and no character a URL would have to escape.
func NormaliseBasePath(s string) (string, error) {
	if s == "" || s == "/" {
		return "/", nil
	}
	trimmed := strings.TrimSuffix(strings.TrimPrefix(s, "/"), "/")
	for _, seg := range strings.Split(trimmed, "/") {
		if seg == "." || seg == ".." || !segmentRe.MatchString(seg) {
			return "", fmt.Errorf("site: the base path %q is not plain path segments (letters, digits and . _ ~ -, separated by single slashes)", s)
		}
	}
	return "/" + trimmed + "/", nil
}

// sitePath places the root-absolute site path p ("/record/") under base, a
// path NormaliseBasePath returned. At the root it returns p unchanged, so a
// build that names no base path writes the bytes it always wrote.
func sitePath(base, p string) string {
	if base == "" || base == "/" {
		return p
	}
	return base + strings.TrimPrefix(p, "/")
}
