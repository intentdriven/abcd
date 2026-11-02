package site

// A project site's content policy, carried in its pages (iss-2610142036270003).
//
// The composed route serves its security headers through `_headers`, which
// Cloudflare reads and serves as response headers. GitHub Pages, the host the
// project route publishes to, reads no such file and serves no response header
// a site declares: a `_headers` in a Pages build is served as a plain file and
// protects nothing. So a project build writes no `_headers`, and each page
// carries its content policy itself, as a <meta http-equiv> element in its
// head, standing ahead of every stylesheet and script the page loads, since a
// policy read from the document governs only what is fetched after it.
//
// The policies are the ones the binary's headers file (setupsrc/headers, the
// copy of abcd's own site-src/headers that setup seeds) gives two route
// families: `/` for the about page and `/record/*` for the record explorer.
// They are read from that file, not restated here, so the page policy and the
// header policy cannot drift apart.
//
// What a page cannot carry, and so a project site on GitHub Pages goes without:
//
//   - frame-ancestors, the refusal to be framed. A policy delivered in a page
//     does not honour it (the CSP 3 specification excludes frame-ancestors,
//     report-uri and sandbox from a <meta> policy, and a browser reports the
//     directive as ignored), so it is dropped from the element rather than
//     written where it does nothing.
//   - X-Content-Type-Options: nosniff, a response header with no in-page form,
//     which a stylesheet, a script or record.json could not carry anyway.
//
// The referrer policy the headers declare, strict-origin-when-cross-origin, is
// the one browsers apply when a host serves none, so it needs no element.

import (
	"fmt"
	"strings"
)

// The route families of the binary's headers file whose content policy a
// project page carries.
const (
	policyAbout  = "/"
	policyRecord = "/record/*"
)

// cspHeader is the header whose value a page carries.
const cspHeader = "Content-Security-Policy"

// metaIgnoredDirectives are the directives a policy delivered in a <meta>
// element does not honour (CSP 3, "Delivering a policy via the <meta>
// element").
var metaIgnoredDirectives = map[string]bool{"frame-ancestors": true, "report-uri": true, "sandbox": true}

// projectPolicies reads the content policy each project route family carries
// from the binary's headers file, less what a page cannot enforce. A family
// the file gives no policy is an error: every page must carry one.
func projectPolicies() (map[string]string, error) {
	data, err := setupSources.ReadFile("setupsrc/headers")
	if err != nil {
		return nil, err
	}
	declared := headerPolicies(string(data))
	out := map[string]string{}
	for _, family := range []string{policyAbout, policyRecord} {
		p := pagePolicy(declared[family])
		if p == "" {
			return nil, fmt.Errorf("site: the binary's headers file gives %s no %s, so a project page would carry none", family, cspHeader)
		}
		out[family] = p
	}
	return out, nil
}

// headerPolicies reads a `_headers` source the way the host reads it: a line
// at the left margin opens a block for a path, an indented `Name: value` sets a
// header on it, and a `#` line is a comment. It returns each path's content
// policy.
func headerPolicies(text string) map[string]string {
	out := map[string]string{}
	block := ""
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if line[0] != ' ' && line[0] != '\t' {
			block = trimmed
			continue
		}
		name, value, ok := strings.Cut(trimmed, ":")
		if ok && block != "" && strings.EqualFold(strings.TrimSpace(name), cspHeader) {
			out[block] = strings.TrimSpace(value)
		}
	}
	return out
}

// pagePolicy is a policy as a page can carry it: each directive kept in
// order, less the ones a <meta> policy does not honour.
func pagePolicy(policy string) string {
	var kept []string
	for _, d := range strings.Split(policy, ";") {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		name, _, _ := strings.Cut(d, " ")
		if metaIgnoredDirectives[strings.ToLower(name)] {
			continue
		}
		kept = append(kept, d)
	}
	return strings.Join(kept, "; ")
}

// policyElement is the element a project page carries its policy in; empty
// on the composed route, whose host serves the policy as a header.
func (c *composer) policyElement(family string) string {
	if !c.project {
		return ""
	}
	return `<meta http-equiv="` + cspHeader + `" content="` + escapeAttr(c.policies[family]) + `">` + "\n"
}

// checkContentPolicy is the gate's content-policy half, on the project route:
// every page carries exactly one policy element, in its head, ahead of every
// stylesheet and script, holding the policy the binary gives its family. A
// page outside the about page and the record has no family, and fails.
func (c *checker) checkContentPolicy() {
	if !c.project {
		return
	}
	policies, err := projectPolicies()
	if err != nil {
		c.fail(CheckContentPolicy, homePageRel, "", "%v", err)
		return
	}
	for _, page := range c.res.Pages {
		family := policyRecord
		switch {
		case page == homePageRel:
			family = policyAbout
		case !strings.HasPrefix(page, recordRoutePrefix):
			c.fail(CheckContentPolicy, page, "", "a project site builds the about page and the record, and this page is neither, so no content policy is declared for it")
			continue
		}
		c.checkPagePolicy(page, policies[family])
	}
}

func (c *checker) checkPagePolicy(page, want string) {
	var metas []*htmlNode
	// readLate: the first policy element comes after a stylesheet or script.
	loaded, readLate := false, false
	c.pages[page].Walk(func(n *htmlNode) {
		if n.Kind != htmlElementNode {
			return
		}
		switch {
		case n.Name == "meta" && strings.EqualFold(n.Attr("http-equiv"), cspHeader):
			if len(metas) == 0 {
				readLate = loaded
			}
			metas = append(metas, n)
		case n.Name == "link" || n.Name == "script":
			loaded = true
		}
	})
	switch {
	case len(metas) == 0:
		c.fail(CheckContentPolicy, page, "", "the page carries no content policy; GitHub Pages serves no header a site declares, so the policy must be a "+
			`<meta http-equiv="Content-Security-Policy"> element in the page's head`)
		return
	case len(metas) > 1:
		c.fail(CheckContentPolicy, page, "", "the page carries %d content-policy elements; a browser enforces every one, so it carries exactly one", len(metas))
		return
	}
	m := metas[0]
	if !m.InsideElement("head") {
		c.fail(CheckContentPolicy, page, "", "the content-policy element is outside the page's head, where a browser does not read it")
	}
	if readLate {
		c.fail(CheckContentPolicy, page, "", "the content-policy element comes after a stylesheet or script the page loads, which it then does not govern")
	}
	if got := m.Attr("content"); got != want {
		c.fail(CheckContentPolicy, page, "", "the page carries %s, not the policy the binary declares for it, %s", quote(got), quote(want))
	}
}
