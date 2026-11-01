package site

// The link-targets gate (iss-2610252224215664).
//
// Two things on the site name a page the site build does not write. A
// composed page links into `/docs/`, in the URL shape siteHref assumes the
// docs generator serves (a directory URL per page), and the redirect map,
// copied verbatim from site-src/redirects to `_redirects`, sends every old
// docs URL to a `/docs/` route written by hand. `/docs/` is the docs
// generator's own tree, written into the output after this build, so no gate
// that reads only what this build wrote can see whether either still lands: a
// generator release that moved its URLs, or a docs page that was renamed,
// would break the links with every other gate passing.
//
// So this gate reads the output as it will be served. Every root-absolute URL
// under `<base>docs/` on a rendered page, and every redirect target, must
// name a file in the output directory: the path itself, or for a directory URL
// (one ending in `/`, or a bare name the host serves as one) its index.html.
// A fragment and a query are not part of the file a URL names and are dropped
// before the lookup. A redirect target that is an absolute URL leaves the
// site, and one carrying a placeholder (`:splat`, `:name`) or a wildcard names
// no single file; neither is judged.
//
// A rendered page here is every page this build writes, the verbatim record
// rendering under `record/` included (iss-2610311732391345). The record is
// exempt from the composed-text gates, which judge the words a composer
// chose, but a record's link to a docs page is written through the same
// siteHref as a composed one's, so a docs page that was renamed breaks it
// just the same. This gate judges where a link lands, never its words, so it
// reads the record pages too; the composed-text gates still do not.
//
// The `/docs/` half can only be judged once the docs tree is in the output. A
// build the docs generator has not written into (the site-render gate's, which
// runs no generator) holds no `docs/` directory, and every `/docs/` URL it
// links is then left unjudged and counted in one NOTE, never passed as checked
// and never failed. The site workflow writes the docs tree before it runs this
// gate, and asserts the tree is there before it does.

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// redirectsOutRel is where the build writes the redirect map, and
// redirectsSourceRel the committed file it is copied from, which is where a
// fix goes.
const (
	redirectsOutRel    = "_redirects"
	redirectsSourceRel = "site-src/redirects"
)

// maxRedirectsBytes bounds the redirect map read.
const maxRedirectsBytes = 1 << 20

// checkLinkTargets runs the gate over every rendered page and the redirect
// map.
func (c *checker) checkLinkTargets() error {
	out, err := os.OpenRoot(c.outDir)
	if err != nil {
		return err
	}
	defer out.Close()

	docsPrefix := c.base + docsRoutePrefix
	docsBuilt := false
	if fi, err := out.Stat(strings.TrimSuffix(docsRoutePrefix, "/")); err == nil && fi.IsDir() {
		docsBuilt = true
	}
	underDocs := func(p string) bool {
		return p == strings.TrimSuffix(docsPrefix, "/") || strings.HasPrefix(p, docsPrefix)
	}
	unjudged := 0

	for _, page := range c.res.Pages {
		for _, n := range findElements(c.pages[page], func(*htmlNode) bool { return true }) {
			for _, attr := range basePathAttrs {
				for _, u := range attrURLs(attr, n.Attr(attr)) {
					p := urlPathOf(u)
					if strings.HasPrefix(u, "//") || !underDocs(p) {
						continue
					}
					if !docsBuilt {
						unjudged++
						continue
					}
					if want, ok := c.serves(out, p); !ok {
						c.fail(CheckLinkTargets, page, nearestSource(n),
							"<%s %s=%s> names nothing in the built output: there is no %s",
							n.Name, attr, quote(u), want)
					}
				}
			}
		}
	}

	data, err := fsutil.ReadGuardedInRoot(out, redirectsOutRel, maxRedirectsBytes)
	switch {
	case os.IsNotExist(err):
	case err != nil:
		return err
	default:
		for _, r := range parseRedirects(data) {
			if strings.Contains(r.to, "://") || strings.HasPrefix(r.to, "//") ||
				strings.ContainsAny(r.to, "*:") {
				continue
			}
			p := urlPathOf(r.to)
			if underDocs(p) && !docsBuilt {
				unjudged++
				continue
			}
			if want, ok := c.serves(out, p); !ok {
				c.fail(CheckLinkTargets, redirectsSourceRel, "",
					"line %d redirects %s to %s, which names nothing in the built output: there is no %s",
					r.line, r.from, r.to, want)
			}
		}
	}

	if unjudged > 0 {
		c.note(CheckLinkTargets, docsRoutePrefix,
			"the output holds no docs/ tree, so %d link(s) and redirect target(s) under %s were not judged; write the docs into the output first",
			unjudged, docsPrefix)
	}
	return nil
}

// serves reports whether the output serves the root-absolute path p, and the
// output-relative file it looked for when it does not.
func (c *checker) serves(out *os.Root, p string) (string, bool) {
	var rel string
	switch {
	case p+"/" == c.base:
	case strings.HasPrefix(p, c.base):
		rel = strings.TrimPrefix(p, c.base)
	default:
		return fmt.Sprintf("file for it under the base path %s", c.base), false
	}
	if dec, err := url.PathUnescape(rel); err == nil {
		rel = dec
	}
	if climbs(rel) || strings.Contains(rel, "\\") {
		return "file for a path that climbs out of the site", false
	}
	index := path.Join(rel, "index.html")
	if rel == "" || strings.HasSuffix(rel, "/") {
		return index, regularIn(out, index)
	}
	rel = path.Clean(rel)
	if regularIn(out, rel) || regularIn(out, path.Join(rel, "index.html")) {
		return "", true
	}
	return rel + " or " + path.Join(rel, "index.html"), false
}

// regularIn reports whether rel is a regular file inside the output root.
func regularIn(out *os.Root, rel string) bool {
	fi, err := out.Stat(filepath.FromSlash(rel))
	return err == nil && fi.Mode().IsRegular()
}

// urlPathOf is a URL's path: the URL without its fragment or query.
func urlPathOf(u string) string {
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		return u[:i]
	}
	return u
}

// nearestSource is the data-src span an element's content came from: its own,
// or the nearest ancestor's.
func nearestSource(n *htmlNode) string {
	if s := n.Attr("data-src"); s != "" {
		return s
	}
	for _, a := range n.Ancestors() {
		if s := a.Attr("data-src"); s != "" {
			return s
		}
	}
	return ""
}

// redirectRule is one line of the redirect map: `from to [status]`.
type redirectRule struct {
	line     int
	from, to string
}

// parseRedirects reads the redirect map's rules. A blank line and a `#`
// comment carry none; a line with fewer than two fields is not a rule the
// host can follow and names no target.
func parseRedirects(data []byte) []redirectRule {
	var out []redirectRule
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64<<10), maxRedirectsBytes)
	for n := 1; sc.Scan(); n++ {
		f := strings.Fields(sc.Text())
		if len(f) < 2 || strings.HasPrefix(f[0], "#") {
			continue
		}
		out = append(out, redirectRule{line: n, from: f[0], to: f[1]})
	}
	return out
}
