package ahoy

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

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
// setup hands its bodies to syncReadmeParts (stepReadme, readme_apply.go).

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
		src.notes = nil
		body, addrs, reason := src.derive(name)
		d := readmeDerived{Name: name}
		if reason != "" {
			d.Reason = name + " omitted: " + reason
		} else {
			var checked []string
			d.Body, d.Reason, checked = checks.run(name, body, addrs)
			if d.Reason == "" {
				d.Notes = append(src.notes, checked...)
			}
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

	// notes is what deriving the current part noted without omitting it (a
	// contents entry listed without a link); deriveReadmeParts takes them.
	notes []string
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
	// setextUnderlineRe underlines the paragraph above it into a setext
	// heading: equals signs for level one, hyphens for level two.
	setextUnderlineRe = regexp.MustCompile(`^ {0,3}(=+|-+)[ \t]*$`)
	// thematicBreakRe is a rule: three or more of one of *, - or _, spaced
	// or not. After a paragraph a run of bare hyphens is an underline instead.
	thematicBreakRe = regexp.MustCompile(`^ {0,3}(?:(?:\*[ \t]*){3,}|(?:-[ \t]*){3,}|(?:_[ \t]*){3,})$`)
	// blockStartRe opens a list item, a quotation or an HTML block, none of
	// which is a top-level paragraph an underline could make a heading.
	blockStartRe = regexp.MustCompile(`^ {0,3}(?:[-+*](?:[ \t]|$)|[0-9]{1,9}[.)](?:[ \t]|$)|>|<[A-Za-z/!?])`)
	// indentedCodeRe is a line indented as code where no paragraph is open.
	indentedCodeRe = regexp.MustCompile(`^(?: {4}| {0,3}\t)`)
)

const noLevelTwoHeading = "the README has no level-two heading"

// contents lists the README's live level-two and level-three headings outside
// every part, ATX (`## Install`) and setext (`Install` underlined with
// hyphens) alike. A heading counts where mdrecord.Mask leaves the line live
// (not in a code fence or an HTML comment) and outside a leading frontmatter
// block. The setext reading is the top-level one: a heading inside a list
// item or a quotation is not read. Anchors follow the forge's slug rule, numbered across every heading
// the page renders, so a repeated heading links to the one it names.
func (r *readmeSource) contents() (string, string) {
	data, err := readInRoot(r.root, "README.md")
	if err != nil {
		return "", readmeUnreadReason(err)
	}
	lines, starts := readmePageLines(data)
	mask := mdrecord.Mask(lines)
	skip := make([]bool, len(lines))
	if loc := frontmatterRe.FindIndex(data); loc != nil {
		for i := range lines {
			skip[i] = starts[i] < loc[1]
		}
	}
	// The parts are where the writer finds them; a page line is in one when
	// it starts inside the part's bytes.
	inPart := make([]bool, len(lines))
	doc := parseReadmeParts(data)
	for _, p := range doc.parts {
		from, to := doc.starts[p.Begin], doc.ends[p.End]
		for i := range lines {
			if starts[i] >= from && starts[i] < to {
				inPart[i] = true
			}
		}
	}
	seen := newReadmeSlugger()
	var out []string
	hasH2 := false
	// heading numbers every heading the page renders and lists the live
	// level-two and level-three ones outside every part. slugText is what
	// the forge makes the anchor from; text is what the list shows. A heading
	// whose anchor abcd cannot be sure of is listed without a link, and a
	// note says which and why.
	//
	// Both are masked (mdrecord.MaskCodeSpans), read once from the heading as
	// written and carried through with their unmask: a second reading of the
	// unmasked text would pair two spans a stripped tag had sat between
	// (iss-2610112237411831).
	heading := func(i, level int, slugText, text string, unmask func(string, bool) string) {
		slug, why := seen.anchor(slugText, unmask)
		if inPart[i] || text == "" || (level != 2 && level != 3) {
			return
		}
		// A code span over a setext heading's lines is listed on one line.
		listed := strings.ReplaceAll(escapeLinkText(text, unmask), "\n", " ")
		indent := ""
		if level == 3 {
			indent = "  "
		} else {
			hasH2 = true
		}
		if why != "" {
			out = append(out, indent+"- "+listed)
			r.notes = append(r.notes, fmt.Sprintf("contents part: the heading at line %d is listed without a link: %s", i+1, why))
			return
		}
		out = append(out, indent+"- ["+listed+"](#"+slug+")")
	}
	// A setext heading is the paragraph its underline ends, so the open
	// paragraph's lines are kept, with the line it began on; inBlock is a
	// list item, a quotation or an HTML block, whose lines open no top-level
	// paragraph until a blank line ends it.
	var para []string
	paraAt := -1
	inBlock := false
	// htmlBlock is the HTML block the line is in, nil outside one: every line of
	// one is HTML to the forge, a heading's included (iss-2610111851312171).
	var htmlBlock *readmeHTMLBlock
	for i, ln := range lines {
		if skip[i] || mask[i] != 0 {
			para, inBlock = nil, false
			continue
		}
		blank := strings.TrimSpace(ln) == ""
		if htmlBlock != nil {
			if htmlBlock.ends(ln, blank) {
				htmlBlock = nil
				inBlock = false
			}
			continue
		}
		if b := openReadmeHTMLBlock(ln, para != nil); b != nil {
			para, inBlock = nil, false
			if !b.ends(ln, false) {
				htmlBlock = b
			}
			continue
		}
		if m := atxHeadingRe.FindStringSubmatch(ln); m != nil {
			para, inBlock = nil, false
			// The space a stripped tag leaves before the text is in the
			// anchor, as it is in the forge's.
			text, unmask := headingText(m[2])
			heading(i, len(m[1]), text, strings.TrimSpace(text), unmask)
			continue
		}
		if para != nil {
			if m := setextUnderlineRe.FindStringSubmatch(ln); m != nil {
				level := 2
				if m[1][0] == '=' {
					level = 1
				}
				// The forge's page keeps the line break between a heading's
				// lines, which its anchor drops; the list shows a space.
				joined, unmask := headingText(strings.Join(para, "\n"))
				heading(paraAt, level, joined, strings.TrimSpace(strings.ReplaceAll(joined, "\n", " ")), unmask)
				para = nil
				continue
			}
			switch {
			case blank:
				para = nil
			case thematicBreakRe.MatchString(ln):
				para = nil
			case blockStartRe.MatchString(ln):
				para, inBlock = nil, true
			default:
				para = append(para, strings.TrimSpace(ln))
			}
			continue
		}
		switch {
		case blank:
			inBlock = false
		case inBlock, indentedCodeRe.MatchString(ln), thematicBreakRe.MatchString(ln):
		case blockStartRe.MatchString(ln):
			inBlock = true
		default:
			para, paraAt = []string{strings.TrimSpace(ln)}, i
		}
	}
	if !hasH2 {
		return "", noLevelTwoHeading
	}
	return strings.Join(out, "\n"), ""
}

// readmeHTMLBlock is an open HTML block, as the forge's markdown reads one: a
// block opened by a pre, script, style or textarea tag, or by `<?`, `<!X` or
// `<![CDATA[`, ends at the line holding its end mark; any other ends at the
// first blank line.
type readmeHTMLBlock struct{ end *regexp.Regexp }

// ends reports whether the block ends at this line (a blank line that ends a
// block is not part of it, and is no heading either way).
func (b *readmeHTMLBlock) ends(ln string, blank bool) bool {
	if b.end == nil {
		return blank
	}
	return b.end.MatchString(ln)
}

// The HTML block starts, after up to three spaces of indent.
var (
	htmlRawBlockRe   = regexp.MustCompile(`(?i)^ {0,3}<(?:pre|script|style|textarea)(?:[ \t>]|$)`)
	htmlRawBlockEnd  = regexp.MustCompile(`(?i)</(?:pre|script|style|textarea)>`)
	htmlPIBlockRe    = regexp.MustCompile(`^ {0,3}<\?`)
	htmlPIBlockEnd   = regexp.MustCompile(`\?>`)
	htmlCDATABlockRe = regexp.MustCompile(`^ {0,3}<!\[CDATA\[`)
	htmlCDATAEnd     = regexp.MustCompile(`\]\]>`)
	htmlDeclBlockRe  = regexp.MustCompile(`^ {0,3}<![A-Za-z]`)
	htmlDeclBlockEnd = regexp.MustCompile(`>`)
	// htmlTagBlockRe opens a block on one of the block-level tag names,
	// opening or closing, which may interrupt a paragraph.
	htmlTagBlockRe = regexp.MustCompile(`(?i)^ {0,3}</?(?:address|article|aside|base|basefont|blockquote|body|caption|center|col|colgroup|dd|details|dialog|dir|div|dl|dt|fieldset|figcaption|figure|footer|form|frame|frameset|h[1-6]|head|header|hr|html|iframe|legend|li|link|main|menu|menuitem|nav|noframes|ol|optgroup|option|p|param|search|section|summary|table|tbody|td|tfoot|th|thead|title|tr|track|ul)(?:[ \t>]|/>|$)`)
	// htmlLoneTagRe is any other complete tag alone on its line, which opens
	// a block only outside a paragraph.
	htmlLoneTagRe = regexp.MustCompile(`^ {0,3}(?:` + inlineTagRe.String() + `)[ \t]*$`)
)

// openReadmeHTMLBlock is the HTML block ln opens, or nil. inPara says a
// paragraph is open, which a lone tag of another name cannot interrupt.
func openReadmeHTMLBlock(ln string, inPara bool) *readmeHTMLBlock {
	switch {
	case htmlRawBlockRe.MatchString(ln):
		return &readmeHTMLBlock{end: htmlRawBlockEnd}
	case strings.HasPrefix(strings.TrimLeft(ln, " "), "<!--"):
		// A comment is mdrecord.Mask's; it never reaches here masked out.
		return nil
	case htmlPIBlockRe.MatchString(ln):
		return &readmeHTMLBlock{end: htmlPIBlockEnd}
	case htmlCDATABlockRe.MatchString(ln):
		return &readmeHTMLBlock{end: htmlCDATAEnd}
	case htmlDeclBlockRe.MatchString(ln):
		return &readmeHTMLBlock{end: htmlDeclBlockEnd}
	case htmlTagBlockRe.MatchString(ln):
		return &readmeHTMLBlock{}
	case !inPara && htmlLoneTagRe.MatchString(ln):
		return &readmeHTMLBlock{}
	}
	return nil
}

// readmePageLines splits the README at every line ending the forge's page
// reads, CRLF, LF and a bare CR, returning each line without its ending and
// the byte offset it starts at. markerLines, the writer's reading, ends a
// line at LF only, so a bare CR would otherwise carry the text after it into
// the heading before it, and hide a heading that follows it.
func readmePageLines(data []byte) (lines []string, starts []int) {
	for off := 0; off < len(data); {
		i := bytes.IndexAny(data[off:], "\r\n")
		starts = append(starts, off)
		if i < 0 {
			lines = append(lines, string(data[off:]))
			break
		}
		lines = append(lines, string(data[off:off+i]))
		next := off + i + 1
		if data[off+i] == '\r' && next < len(data) && data[next] == '\n' {
			next++
		}
		off = next
	}
	return lines, starts
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
//
// It also keeps the bases it cannot be sure of. The forge makes an anchor
// from the text it renders (iss-2610111850237083): a character reference is
// decoded first, which abcd does as the forge does, but an emoji shortcode
// becomes an emoji, dropped from the anchor, only when the forge knows its
// name, and abcd holds no copy of the forge's list. A heading carrying one
// has one of two anchors, and so does every later heading whose anchor it
// could have taken or numbered: each of those bases, and any base numbered
// from one, is unsure.
type readmeSlugger struct {
	taken  map[string]int
	unsure []string
}

func newReadmeSlugger() *readmeSlugger { return &readmeSlugger{taken: map[string]int{}} }

// emojiShortcodeRe is a shortcode the forge may render as an emoji.
var emojiShortcodeRe = regexp.MustCompile(`:[a-z0-9_+-]+:`)

// charRefRe is a character reference as the forge decodes one: a named,
// decimal or hexadecimal reference ending in a semicolon.
var charRefRe = regexp.MustCompile(`&(?:#[0-9]{1,7}|#[xX][0-9A-Fa-f]{1,6}|[A-Za-z][A-Za-z0-9]{0,31});`)

// decodeCharRefs decodes every character reference the HTML entity table
// knows, and leaves an unknown one as written, as the forge's page does. The
// table's decoder also reads a known name's prefix (`&notanentity;` as `&not`
// and the rest), so a decoding is taken only when it consumed the whole
// reference: every entity stands for at most two characters. s is masked
// (mdrecord.MaskCodeSpans): a reference in a code span is text, never decoded.
func decodeCharRefs(s string) string {
	return charRefRe.ReplaceAllStringFunc(s, func(ref string) string {
		if dec := html.UnescapeString(ref); dec != ref && utf8.RuneCountInString(dec) <= 2 {
			return dec
		}
		return ref
	})
}

// Why a heading is listed without a link.
const (
	readmeAnchorShortcode = "it carries an emoji shortcode, and which anchor the forge gives it depends on whether the forge knows the shortcode, which abcd cannot tell"
	readmeAnchorNumbered  = "its anchor depends on how the forge anchors an earlier heading carrying an emoji shortcode"
)

// anchor returns the next heading's anchor and records it as taken, or, when
// abcd cannot be sure of the anchor, the reason it is not given.
//
// A code span is read as the text it renders: a reference or a shortcode in
// one is text, and the slug is made from the span's content. masked is the
// heading's text with its code spans masked, as headingText returns it, and
// unmask the function that restores them.
func (s *readmeSlugger) anchor(masked string, unmask func(string, bool) string) (string, string) {
	masked = decodeCharRefs(masked)
	why := ""
	if emojiShortcodeRe.MatchString(masked) {
		why = readmeAnchorShortcode
		s.unsure = append(s.unsure, headingSlug(unmask(emojiShortcodeRe.ReplaceAllString(masked, ""), true)), headingSlug(unmask(masked, true)))
	}
	base := headingSlug(unmask(masked, true))
	if why == "" && s.isUnsure(base) {
		why = readmeAnchorNumbered
	}
	slug := base
	for {
		if _, taken := s.taken[slug]; !taken {
			break
		}
		s.taken[base]++
		slug = fmt.Sprintf("%s-%d", base, s.taken[base])
	}
	s.taken[slug] = 0
	return slug, why
}

// isUnsure reports whether base is, or is numbered from, a base abcd cannot
// be sure of.
func (s *readmeSlugger) isUnsure(base string) bool {
	for _, u := range s.unsure {
		if base == u || strings.HasPrefix(base, u+"-") {
			return true
		}
	}
	return false
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

// inlineTagRe is inline HTML as the forge reads it in a heading: an opening
// tag with its attributes, a closing tag, or a comment. An autolink
// (`<https://…>`) and a lone angle bracket are not tags.
var inlineTagRe = regexp.MustCompile(`<[A-Za-z][A-Za-z0-9-]*(?:\s+[A-Za-z_:][A-Za-z0-9_.:-]*(?:\s*=\s*(?:[^\s"'=<>` + "`" + `]+|'[^']*'|"[^"]*"))?)*\s*/?>|</[A-Za-z][A-Za-z0-9-]*\s*>|<!--.*?-->`)

// headingText is a heading's text as the page shows it: an inline link read
// as its text, and inline HTML tags dropped, which render as markup
// (iss-2610111850405447). The spaces around a dropped tag are kept. A code
// span shows its text as written, so neither reaches into one; the span is
// kept with its backticks, which the list shows as code.
//
// The text comes back masked, with the function that restores its spans: the
// spans are read once, from the heading as written (iss-2610112237411831).
func headingText(src string) (string, func(string, bool) string) {
	masked, unmask := mdrecord.MaskCodeSpans(src)
	return inlineTagRe.ReplaceAllString(inlineLinkRe.ReplaceAllString(masked, "$1"), ""), unmask
}

// spanJoinRe is a masked code span with a backtick or another span directly
// after it, or a backtick directly before it: written out side by side, the
// backtick runs would merge and a reader would pair them differently.
var spanJoinRe = regexp.MustCompile("(\x00[0-9]+\x00)(\x00|`)|(`)(\x00[0-9]+\x00)")

// escapeLinkText keeps a heading's text from closing the link it sits in, and
// restores its code spans as written. A bracket in a code span closes nothing
// and a backslash there is literal, so a span is left as written. Where a
// stripped tag or link left a span touching another span or a backtick, a
// space keeps them apart, as the list shows a setext heading's line break.
func escapeLinkText(masked string, unmask func(string, bool) string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `[`, `\[`, `]`, `\]`).Replace(masked)
	for spanJoinRe.MatchString(escaped) {
		escaped = spanJoinRe.ReplaceAllString(escaped, "$1$3 $2$4")
	}
	return unmask(escaped, false)
}
