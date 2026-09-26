package lint

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// ruleLinkAnchors validates a link's #fragment against the headings of the page
// it names (iss-303). links_resolve strips the fragment before resolving the
// file and skips a same-file #link outright, so a heading anchor was checked by
// nothing; this rule is its own id so it can land warn-first beside a blocking
// links_resolve, and read the same `exempt` globs for the same mirror files.
const ruleLinkAnchors = "link_anchors"

var (
	// anchorHeadingRe is an ATX heading and its text, closing hashes dropped.
	anchorHeadingRe = regexp.MustCompile(`^ {0,3}#{1,6}[ \t]+(.*?)(?:[ \t]+#+)?[ \t]*$`)
	// htmlAnchorRe is an explicit HTML anchor a fragment may name.
	htmlAnchorRe = regexp.MustCompile(`<a\s+(?:[^>]*\s)?(?:id|name)\s*=\s*["']([^"']+)["']`)
	// inlineLinkTextRe keeps a heading link's text and drops its target, as the
	// rendered heading does.
	inlineLinkTextRe = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
)

// headingSlug is GitHub's heading anchor: the rendered text lower-cased, every
// character that is not a letter, a mark, a number, a hyphen, an underscore or a
// space removed, and each space turned into a hyphen.
func headingSlug(text string) string {
	text = inlineLinkTextRe.ReplaceAllString(text, "$1")
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

// pageAnchors is the set of fragments a markdown page answers to: its ATX
// heading slugs outside fences, a repeated slug suffixed -1, -2 in order as
// GitHub numbers them, and the ids and names of its explicit HTML anchors.
func pageAnchors(lines []string) map[string]bool {
	mask := fenceMask(lines)
	seen := map[string]int{}
	out := map[string]bool{}
	for i, line := range lines {
		if mask[i] {
			continue
		}
		for _, m := range htmlAnchorRe.FindAllStringSubmatch(line, -1) {
			out[strings.ToLower(m[1])] = true
		}
		m := anchorHeadingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		slug := headingSlug(m[1])
		n := seen[slug]
		seen[slug] = n + 1
		if n > 0 {
			slug += "-" + strconv.Itoa(n)
		}
		out[slug] = true
	}
	return out
}

// checkLinkAnchors reports each link whose fragment names no anchor of the
// markdown page it resolves to (the linking page itself for a bare #fragment).
// A target that does not resolve is links_resolve's finding, and a target that
// is not a markdown file has no headings to hold a fragment to, so both are
// silent here. slugs caches each target's anchors across the walk.
func checkLinkAnchors(rel, fileAbs, repoRoot string, lines []string, mask []bool, cfg RuleConfig, slugs map[string]map[string]bool) []Finding {
	fileDir := filepath.Dir(fileAbs)
	var out []Finding
	for i, line := range lines {
		if mask[i] {
			continue
		}
		for _, m := range linkRe.FindAllStringSubmatch(stripInlineCode(line), -1) {
			target := strings.TrimSpace(m[1])
			hash := strings.IndexByte(target, '#')
			if hash < 0 || strings.HasPrefix(target, "//") || schemeRe.MatchString(target) {
				continue
			}
			fragment := target[hash+1:]
			if fragment == "" {
				continue
			}
			if dec, err := url.PathUnescape(fragment); err == nil {
				fragment = dec
			}
			path := target[:hash]
			if q := strings.IndexByte(path, '?'); q >= 0 {
				path = path[:q]
			}
			pageAbs := fileAbs
			if path != "" {
				pageAbs = filepath.Join(fileDir, path)
			}
			anchors, ok := slugs[pageAbs]
			if !ok {
				anchors = readPageAnchors(repoRoot, pageAbs, fileAbs, lines)
				slugs[pageAbs] = anchors
			}
			if anchors == nil || anchors[strings.ToLower(fragment)] {
				continue
			}
			out = append(out, Finding{
				File: rel, Line: i + 1, RuleID: ruleLinkAnchors, Severity: cfg.Severity,
				Message: "link fragment names no heading or anchor of its target page: " + m[1],
			})
		}
	}
	return out
}

// readPageAnchors reads one target page's anchors, or nil when the page has
// none to check against: not markdown, not inside the repository, or not a
// readable regular file (whether it resolves at all is links_resolve's).
func readPageAnchors(repoRoot, pageAbs, fileAbs string, lines []string) map[string]bool {
	if pageAbs == fileAbs {
		return pageAnchors(lines)
	}
	if !hasMarkdownExt(pageAbs) {
		return nil
	}
	realPath, err := containedRealPath(repoRoot, pageAbs)
	if err != nil {
		return nil
	}
	if st, err := os.Stat(realPath); err != nil || !st.Mode().IsRegular() {
		return nil
	}
	content, err := fsutil.ReadGuarded(realPath, citationPageSizeLimit)
	if err != nil {
		return nil
	}
	return pageAnchors(strings.Split(string(content), "\n"))
}
