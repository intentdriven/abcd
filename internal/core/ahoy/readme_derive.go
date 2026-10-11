package ahoy

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Deriving each README part from the project on this computer
// (spc-2610031704338417 step 2, for itd-2610031348087517).
//
// deriveReadmeParts takes a root directory and the stored settings and
// returns, per chosen part in the fixed order, either the body to write or the
// reason it is omitted. Every body then passes the three checks in
// readme_check.go (the badge-address check, the outbound scrub and the
// project's own docs-lint), and a part failing any one is omitted naming the
// check's finding.
//
// Everything is read from files under root and from `git remote get-url
// origin` (originURL). This file imports no network package: net/url is used
// only as a parser, and the live badges are drawn by the forge when someone
// reads the page (scope condition 5).
//
// Staged: nothing outside the tests calls deriveReadmeParts yet. The spec's
// step 3 hands its bodies to syncReadmeParts from setup.

// readmeSettings is what derivation reads of the stored settings: the chosen
// parts (`readme.parts`) and the owner's declared visibility
// (`repo.visibility`), the only source of "public" (scope condition 2).
type readmeSettings struct {
	Parts      []string
	Visibility string
}

// public reports whether the owner declared the project public.
func (s readmeSettings) public() bool { return s.Visibility == "public" }

// readmeDerived is one chosen part's derivation: a body that passed every
// check, or the reason it is omitted (each reason leads with the part's name).
// Notes carry what a check noted without refusing (a docs-lint warning).
type readmeDerived struct {
	Name   string
	Body   string
	Reason string
	Notes  []string
}

// readmeAddr is one address a badge part carries, with the role that decides
// which host the badge-address check admits for it.
type readmeAddr struct {
	Role string
	URL  string
}

const (
	readmeAddrLink  = "link"
	readmeAddrImage = "image"
)

// deriveReadmeParts derives every chosen part, in the fixed order, and runs
// the three checks on each body. A name outside the fixed order is reported
// omitted; a name chosen twice is derived once.
func deriveReadmeParts(root string, s readmeSettings) []readmeDerived {
	chosen := map[string]bool{}
	var out []readmeDerived
	for _, name := range s.Parts {
		if readmePartIndex(name) < 0 && !chosen[name] {
			out = append(out, readmeDerived{Name: name, Reason: fmt.Sprintf("%q omitted: not a part abcd writes (known: %s)", name, strings.Join(readmePartOrder, ", "))})
		}
		chosen[name] = true
	}
	src := &readmeSource{root: root, settings: s}
	checks := newReadmeChecks(root)
	var derived []readmeDerived
	for _, name := range readmePartOrder {
		if !chosen[name] {
			continue
		}
		body, addrs, reason := src.derive(name)
		d := readmeDerived{Name: name}
		if reason != "" {
			d.Reason = name + " omitted: " + reason
		} else {
			d.Body, d.Reason, d.Notes = checks.run(name, body, addrs)
		}
		derived = append(derived, d)
	}
	return append(derived, out...)
}

// readmeSource reads what the parts are derived from, each source at most once
// per derivation.
type readmeSource struct {
	root     string
	settings readmeSettings

	originRead bool
	origin     forgeRemote
	originWhy  string
}

func (r *readmeSource) derive(name string) (body string, addrs []readmeAddr, reason string) {
	switch name {
	case "release-badge":
		return r.releaseBadge()
	case "build-badge":
		return r.buildBadge()
	case "abcd-badge":
		return abcdBadge()
	case "licence":
		body, reason = r.licence()
	case "links":
		body, reason = r.links()
	case "contents":
		body, reason = r.contents()
	}
	return body, nil, reason
}

// ---------------------------------------------------------------------------
// The badges
// ---------------------------------------------------------------------------

// readmeForgeHost is the one forge abcd composes badges for (open question 3:
// github.com only); readmeBadgeImageHost draws the release and abcd badges'
// images (open question 1), the forge drawing no release badge of its own.
const (
	readmeForgeHost      = "github.com"
	readmeBadgeImageHost = "img.shields.io"
	abcdPublicRepo       = "https://github.com/intentdriven/abcd"
)

// forgeRemote is the origin remote read as a forge address: its host, owner
// and repository, and any query string it carried. The derivation composes
// every forge address from it unaltered, host and query included, so a remote
// carrying a foreign host or query is exactly what the badge-address check
// meets and names.
type forgeRemote struct {
	Host, Owner, Name, RawQuery string
}

// forgeRemote reads the origin once. Its reason names no part of the remote
// beyond what a refusal needs: the URL is repository-controlled input.
func (r *readmeSource) forgeRemote() (forgeRemote, string) {
	if !r.originRead {
		r.originRead = true
		r.origin, r.originWhy = parseForgeRemote(originURL(r.root))
	}
	return r.origin, r.originWhy
}

// scpRemoteRe is git's scp-like remote form, `user@host:owner/repo`.
var scpRemoteRe = regexp.MustCompile(`^[A-Za-z0-9._-]+@([A-Za-z0-9.-]+):([^/].*)$`)

// parseForgeRemote reads an origin URL in the forms git writes: https or http,
// ssh:// and git://, and the scp-like `git@host:owner/repo`. The host is kept
// as written (lower-cased), and so is a query string; the path must name
// exactly one owner and one repository, each a plain name.
func parseForgeRemote(origin string) (forgeRemote, string) {
	if origin == "" {
		return forgeRemote{}, "no origin remote"
	}
	var host, path, query string
	if m := scpRemoteRe.FindStringSubmatch(origin); m != nil && !strings.Contains(origin, "://") {
		host, path = m[1], m[2]
		if i := strings.IndexByte(path, '?'); i >= 0 {
			path, query = path[:i], path[i+1:]
		}
	} else {
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			return forgeRemote{}, "the origin remote is not an address abcd can read"
		}
		switch u.Scheme {
		case "https", "http":
			host = u.Host
		case "ssh", "git":
			// The port belongs to the transport, not to the forge's pages.
			host = u.Hostname()
		default:
			return forgeRemote{}, "the origin remote is not an address abcd can read"
		}
		path, query = u.Path, u.RawQuery
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return forgeRemote{}, "the origin remote does not name exactly one owner and one repository"
	}
	if !ownerRepoRe.MatchString(parts[0]) || !ownerRepoRe.MatchString(parts[1]) {
		return forgeRemote{}, "the origin remote's owner or repository is not a plain name"
	}
	return forgeRemote{Host: strings.ToLower(host), Owner: parts[0], Name: parts[1], RawQuery: query}, ""
}

// page is the forge address of one page of the repository, carrying the
// remote's query string as the remote does.
func (f forgeRemote) page(rest string) string {
	u := "https://" + f.Host + "/" + f.Owner + "/" + f.Name + rest
	if f.RawQuery != "" {
		u += "?" + f.RawQuery
	}
	return u
}

// forgeBadgeRemote is the remote a forge badge is composed from, or the
// reason there is none: the owner did not declare the project public, or the
// origin cannot be read.
func (r *readmeSource) forgeBadgeRemote() (forgeRemote, string) {
	if !r.settings.public() {
		vis := termsafe.Sanitize(r.settings.Visibility)
		if vis == "" {
			vis = "not set"
		}
		return forgeRemote{}, "not declared public (repo.visibility in .abcd/config.json is " + vis + "; abcd reads no visibility from the forge)"
	}
	return r.forgeRemote()
}

// releaseBadge is the latest-release badge, drawn from the forge's data so no
// version is written into the file (decision 4), linked to the forge's
// latest-release page.
func (r *readmeSource) releaseBadge() (string, []readmeAddr, string) {
	f, why := r.forgeBadgeRemote()
	if why != "" {
		return "", nil, why
	}
	img := "https://" + readmeBadgeImageHost + "/github/v/release/" + f.Owner + "/" + f.Name
	link := f.page("/releases/latest")
	return badgeMarkdown("Latest release", img, link), []readmeAddr{{readmeAddrImage, img}, {readmeAddrLink, link}}, ""
}

// buildBadge is the forge's own workflow status badge, linked to the
// workflow's runs.
func (r *readmeSource) buildBadge() (string, []readmeAddr, string) {
	f, why := r.forgeBadgeRemote()
	if why != "" {
		return "", nil, why
	}
	wf, why := r.workflow()
	if why != "" {
		return "", nil, why
	}
	img := f.page("/actions/workflows/" + wf + "/badge.svg")
	link := f.page("/actions/workflows/" + wf)
	return badgeMarkdown("Build", img, link), []readmeAddr{{readmeAddrImage, img}, {readmeAddrLink, link}}, ""
}

// abcdBadge is the opt-in managed-with-abcd badge (decision 2). It reads
// nothing from the project, so it is never omitted once chosen.
func abcdBadge() (string, []readmeAddr, string) {
	img := "https://" + readmeBadgeImageHost + "/badge/managed%20with-abcd-blue"
	return badgeMarkdown("Managed with abcd", img, abcdPublicRepo), []readmeAddr{{readmeAddrImage, img}, {readmeAddrLink, abcdPublicRepo}}, ""
}

func badgeMarkdown(alt, img, link string) string {
	return "[![" + alt + "](" + img + ")](" + link + ")"
}

// workflowFileRe is a workflow file name the badge address can carry as it
// stands; any other name in the folder is not offered.
var workflowFileRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.ya?ml$`)

// workflow names the one workflow the build badge shows (open question 2):
// the only workflow file in .github/workflows/, else ci.yml or ci.yaml among
// several, else none, naming the files found.
func (r *readmeSource) workflow() (string, string) {
	var found []string
	for _, n := range regularFilesIn(r.root, ".github/workflows") {
		if workflowFileRe.MatchString(n) {
			found = append(found, n)
		}
	}
	switch len(found) {
	case 0:
		return "", "no workflow file in .github/workflows/"
	case 1:
		return found[0], ""
	}
	for _, want := range []string{"ci.yml", "ci.yaml"} {
		for _, n := range found {
			if n == want {
				return n, ""
			}
		}
	}
	const shown = 8
	list := found
	more := ""
	if len(list) > shown {
		list, more = list[:shown], fmt.Sprintf(" and %d more", len(found)-shown)
	}
	return "", fmt.Sprintf("several workflows in .github/workflows/ and none is ci.yml or ci.yaml: %s%s", strings.Join(list, ", "), more)
}

// regularFilesIn lists the regular files directly in root/rel ("" for the
// root itself) by their exact names, sorted. The folder is opened inside root
// (os.Root), so a symlinked component cannot walk the listing out of the
// project, and it must itself be a real directory. A name is matched as the
// directory holds it, never through a case-insensitive lookup, so a link the
// forge serves names a file that is there; a symlink is not a file the
// project has at that place.
func regularFilesIn(root, rel string) []string {
	rt, err := os.OpenRoot(root)
	if err != nil {
		return nil
	}
	defer rt.Close()
	if rel == "" {
		rel = "."
	} else if fi, err := rt.Lstat(rel); err != nil || !fi.IsDir() {
		return nil
	}
	dir, err := rt.Open(rel)
	if err != nil {
		return nil
	}
	defer dir.Close()
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.Type().IsRegular() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

func hasRegularFile(root, rel, name string) bool {
	for _, n := range regularFilesIn(root, rel) {
		if n == name {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// The licence
// ---------------------------------------------------------------------------

// licenceFiles are looked for at the root, in this order.
var licenceFiles = []string{"LICENSE", "LICENSE.md", "LICENSE.txt"}

// licenceNames is the one table a licence is named from: an identifier's
// display name, for the identifiers the forge-standard first lines map to.
var licenceNames = map[string]string{
	"MIT":          "the MIT License",
	"Apache-2.0":   "the Apache License 2.0",
	"BSD-2-Clause": "the BSD 2-Clause License",
	"BSD-3-Clause": "the BSD 3-Clause License",
	"ISC":          "the ISC License",
	"MPL-2.0":      "the Mozilla Public License 2.0",
}

// licenceFirstLines maps a forge-standard first line to its identifier. The
// GNU licences are not here: their text alone does not say whether "or any
// later version" applies, so they are named only from an identifier line.
// Apache's first line is matched apart, with the line after it.
var licenceFirstLines = map[string]string{
	"MIT License":                        "MIT",
	"BSD 2-Clause License":               "BSD-2-Clause",
	"BSD 3-Clause License":               "BSD-3-Clause",
	"ISC License":                        "ISC",
	"Mozilla Public License Version 2.0": "MPL-2.0",
}

var (
	// spdxLineRe finds an identifier line, behind any comment opener.
	spdxLineRe = regexp.MustCompile(`SPDX-License-Identifier:[ \t]*(.*?)[ \t]*(?:\*/|-->)?[ \t]*$`)
	// spdxExprRe is the identifier expressions abcd will repeat: identifiers
	// joined by AND, OR or WITH, with parentheses.
	spdxExprRe = regexp.MustCompile(`^[A-Za-z0-9.+()-]+(?: (?:AND|OR|WITH) [A-Za-z0-9.+()-]+)*$`)
)

// spdxIdentifierLines is how far into the file an identifier line counts.
const spdxIdentifierLines = 20

func (r *readmeSource) licence() (string, string) {
	file := ""
	for _, n := range licenceFiles {
		if hasRegularFile(r.root, "", n) {
			file = n
			break
		}
	}
	if file == "" {
		return "", "no LICENSE file (LICENSE, LICENSE.md or LICENSE.txt) at the root"
	}
	link := "[" + file + "](" + file + ")"
	if id := licenceIdentifier(r.root, file); id != "" {
		name := id
		if n, ok := licenceNames[id]; ok {
			name = n
		}
		return "Licensed under " + name + "; see " + link + ".", ""
	}
	return "See " + link + " for the licence terms.", ""
}

// licenceIdentifier names the licence from an identifier line in the file's
// first twenty lines, else from a forge-standard first line, else not at all.
// No licence text is matched as a whole.
func licenceIdentifier(root, file string) string {
	data, err := readInRoot(root, file)
	if err != nil {
		return ""
	}
	data = bytes.TrimPrefix(data, []byte("\ufeff"))
	lines, _, _ := markerLines(data)
	for i, ln := range lines {
		if i >= spdxIdentifierLines {
			break
		}
		if m := spdxLineRe.FindStringSubmatch(ln); m != nil {
			if spdxExprRe.MatchString(m[1]) && len(m[1]) <= 128 {
				return m[1]
			}
			break
		}
	}
	var nonBlank []string
	for _, ln := range lines {
		if t := strings.TrimSpace(ln); t != "" {
			nonBlank = append(nonBlank, t)
			if len(nonBlank) == 2 {
				break
			}
		}
	}
	if len(nonBlank) == 0 {
		return ""
	}
	if id, ok := licenceFirstLines[nonBlank[0]]; ok {
		return id
	}
	if nonBlank[0] == "Apache License" && len(nonBlank) == 2 && strings.HasPrefix(nonBlank[1], "Version 2.0") {
		return "Apache-2.0"
	}
	return ""
}

// readInRoot reads root/rel guarded and contained in root: no symlinked
// component, no non-regular leaf, nothing past maxAhoyFileBytes.
func readInRoot(root, rel string) ([]byte, error) {
	rt, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer rt.Close()
	return fsutil.ReadGuardedInRoot(rt, rel, maxAhoyFileBytes)
}

// ---------------------------------------------------------------------------
// The links
// ---------------------------------------------------------------------------

// readmeLinkFiles are the files the links part looks for, in its order, each
// with the words it is linked by.
var readmeLinkFiles = []struct{ File, Label string }{
	{"CHANGELOG.md", "Changelog"},
	{"CONTRIBUTING.md", "Contributing guide"},
	{"SECURITY.md", "Security policy"},
	{"CODE_OF_CONDUCT.md", "Code of conduct"},
}

// readmeLinkPlaces are where each file is looked for, first found wins.
var readmeLinkPlaces = []string{"", ".github", "docs"}

func (r *readmeSource) links() (string, string) {
	var lines []string
	for _, lf := range readmeLinkFiles {
		for _, place := range readmeLinkPlaces {
			if hasRegularFile(r.root, place, lf.File) {
				target := lf.File
				if place != "" {
					target = place + "/" + lf.File
				}
				lines = append(lines, "- ["+lf.Label+"]("+target+")")
				break
			}
		}
	}
	if len(lines) == 0 {
		return "", "none of CHANGELOG.md, CONTRIBUTING.md, SECURITY.md or CODE_OF_CONDUCT.md is at the root, in .github/ or in docs/"
	}
	return strings.Join(lines, "\n"), ""
}

// ---------------------------------------------------------------------------
// The contents
// ---------------------------------------------------------------------------

var (
	// atxHeadingRe is a live ATX heading: up to three spaces, one to six #s,
	// then a space or the end, the text, and an optional closing run of #s.
	atxHeadingRe = regexp.MustCompile(`^ {0,3}(#{1,6})(?:[ \t]+(.*?))?(?:[ \t]+#+)?[ \t]*$`)
	// inlineLinkRe is an inline link, read as its text, as the page renders it.
	inlineLinkRe = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
)

const noLevelTwoHeading = "the README has no level-two heading"

// contents lists the README's live level-two and level-three headings outside
// every part. A heading counts where mdrecord.Mask leaves the line live (not
// in a code fence or an HTML comment) and outside a leading frontmatter
// block. Anchors follow the forge's slug rule, numbered across every heading
// the page renders, so a repeated heading links to the one it names.
func (r *readmeSource) contents() (string, string) {
	data, err := readInRoot(r.root, "README.md")
	if err != nil {
		return "", readmeUnreadReason(err)
	}
	lines, starts, _ := markerLines(data)
	mask := mdrecord.Mask(lines)
	skip := make([]bool, len(lines))
	if loc := frontmatterRe.FindIndex(data); loc != nil {
		for i := range lines {
			skip[i] = starts[i] < loc[1]
		}
	}
	inPart := make([]bool, len(lines))
	for _, p := range parseReadmeParts(data).parts {
		for i := p.Begin; i <= p.End; i++ {
			inPart[i] = true
		}
	}
	seen := readmeSlugger{}
	var out []string
	hasH2 := false
	for i, ln := range lines {
		if skip[i] || mask[i] != 0 {
			continue
		}
		m := atxHeadingRe.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		text := strings.TrimSpace(inlineLinkRe.ReplaceAllString(m[2], "$1"))
		slug := seen.anchor(text)
		level := len(m[1])
		if inPart[i] || text == "" || (level != 2 && level != 3) {
			continue
		}
		indent := ""
		if level == 3 {
			indent = "  "
		} else {
			hasH2 = true
		}
		out = append(out, indent+"- ["+escapeLinkText(text)+"](#"+slug+")")
	}
	if !hasH2 {
		return "", noLevelTwoHeading
	}
	return strings.Join(out, "\n"), ""
}

// readmeUnreadReason says why the contents part has no README to read. Only
// an absent README has no level-two heading; one abcd will not or cannot read
// is said to be unread, so the reason never claims what the file holds.
func readmeUnreadReason(err error) string {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return noLevelTwoHeading
	case errors.Is(err, fsutil.ErrNotRegular):
		return "README.md is a symlink or not a regular file, so abcd does not read it"
	case errors.Is(err, fsutil.ErrTooBig):
		return fmt.Sprintf("README.md is past abcd's %d-byte cap for the files it reads, so abcd does not read it", maxAhoyFileBytes)
	}
	return "README.md could not be read: " + termsafe.Sanitize(err.Error())
}

// readmeSlugger numbers heading anchors across a page as the forge does: a
// slug already taken takes the next number its own base has not handed out
// and no heading already owns, so `Install`, `Install-1`, `Install` anchor
// #install, #install-1 and #install-2. It counts per base slug.
type readmeSlugger map[string]int

// anchor returns the next heading's anchor and records it as taken.
func (s readmeSlugger) anchor(text string) string {
	base := headingSlug(text)
	slug := base
	for {
		if _, taken := s[slug]; !taken {
			break
		}
		s[base]++
		slug = fmt.Sprintf("%s-%d", base, s[base])
	}
	s[slug] = 0
	return slug
}

// headingSlug is the forge's heading anchor: lower case, every character but
// a letter, a mark, a digit, a hyphen, an underscore or a space dropped, and
// each space a hyphen.
func headingSlug(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeLinkText keeps a heading's text from closing the link it sits in.
func escapeLinkText(s string) string {
	return strings.NewReplacer(`\`, `\\`, `[`, `\[`, `]`, `\]`).Replace(s)
}
