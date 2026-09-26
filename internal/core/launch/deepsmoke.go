package launch

// deepsmoke.go — the installability smoke's deep tier (itd-66 AC4;
// spc-2609201955277614 piece 2).
//
// The light tier (smoke.go) proves every declared path is CARRIED. The deep
// tier proves every declared page LOADS: over the same resolved surface
// (installsurface.go), it renders each command page's, skill's and agent's help
// and frontmatter, so a page that resolves on disk but that a harness would
// fail to load is caught before the tag. The original criterion's
// Python-import clause is moot — nothing shipped imports — and is recorded so
// on the intent.
//
// The rendering runs in an ISOLATED SUBPROCESS rooted at a materialised
// payload: the PageRunner seam is the only way this package reaches one, and
// the front door supplies the runner (the abcd binary itself, re-executed with
// its working directory and HOME inside the throwaway tree and a minimal
// environment). Whatever a page read could touch lands in that tree, which the
// caller removes. A runner that cannot answer, or answers for fewer pages than
// it was asked about, fails the tier: an unanswered page is not a loaded one.

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// SmokeTierDeep is the light tier's assertions plus every declared page's help
// rendered in an isolated subprocess.
const SmokeTierDeep SmokeTier = "deep"

// Deep-tier finding kinds.
const (
	findingPageUnloadable       = "page-unloadable"
	findingDeepSmokeUnavailable = "deep-smoke-unavailable"
)

// maxPageBytes bounds one page read by the deep tier.
const maxPageBytes = 1 << 20

// PageRef is one page the deep tier asks the runner to render.
type PageRef struct {
	Kind SurfaceKind `json:"kind"`
	Path string      `json:"path"`
}

// PageHelp is one rendered page: its help, or why it would not load.
type PageHelp struct {
	Kind         SurfaceKind `json:"kind"`
	Path         string      `json:"path"`
	Name         string      `json:"name,omitempty"`
	Description  string      `json:"description,omitempty"`
	ArgumentHint string      `json:"argument_hint,omitempty"`
	Error        string      `json:"error,omitempty"`
}

// PageRunner renders pages in an isolated subprocess rooted at root, one
// answer per page.
type PageRunner func(root string, pages []PageRef) ([]PageHelp, error)

// DeepSmokeReport is one run of the deep tier.
type DeepSmokeReport struct {
	Tier SmokeTier `json:"tier"`
	OK   bool      `json:"ok"`
	// Checked counts the pages asked about, so a pass over a payload with no
	// pages is visibly vacuous.
	Checked  int            `json:"checked"`
	Pages    []PageHelp     `json:"pages,omitempty"`
	Findings []SmokeFinding `json:"findings,omitempty"`
}

// RenderPageHelp renders one page's help from the tree at root, the way the
// deep tier's subprocess does. A page loads when it is readable UTF-8, any
// frontmatter block it opens is closed and is a mapping with no duplicated key,
// and it renders some help: a description, or for a command or an agent the
// first line of its body. A skill needs a name and a description in its
// frontmatter.
//
// The tier reads frontmatter only to judge whether a harness would load the
// page, so it must never be stricter than the YAML it judges: every shape a
// YAML reader loads as a top-level mapping loads here too — a scalar continued
// on indented lines, a quoted or non-ASCII key, a block closed by YAML's own
// document end (iss-2609251902438821).
func RenderPageHelp(root string, ref PageRef) PageHelp {
	help := PageHelp{Kind: ref.Kind, Path: ref.Path}
	fail := func(format string, args ...any) PageHelp {
		help.Error = fmt.Sprintf(format, args...)
		return help
	}
	if !fsutil.ValidRelPath(ref.Path) {
		return fail("not readable: the path is not inside the payload")
	}
	data, err := fsutil.ReadGuarded(filepath.Join(root, filepath.FromSlash(ref.Path)), maxPageBytes)
	if err != nil {
		return fail("not readable: %v", pathFreeError(err))
	}
	if !utf8.Valid(data) {
		return fail("is not valid UTF-8")
	}
	text := frontmatter.TrimBOM(string(data))
	body := text
	fields := map[string]string{}
	if lines := strings.SplitN(text, "\n", 2); frontmatter.IsDelimiter(lines[0]) {
		inner, rest, closed := pageFrontmatter(text)
		if !closed {
			return fail("opens a frontmatter block that is never closed")
		}
		body = rest
		var reason string
		fields, reason = pageFields(inner)
		if reason != "" {
			return fail("%s", reason)
		}
	}

	help.Name = fields["name"]
	if help.Name == "" {
		if ref.Kind == SurfaceSkill {
			help.Name = path.Base(path.Dir(ref.Path))
		} else {
			help.Name = strings.TrimSuffix(path.Base(ref.Path), ".md")
		}
	}
	help.Description = fields["description"]
	help.ArgumentHint = fields["argument-hint"]
	if ref.Kind == SurfaceSkill && (fields["name"] == "" || fields["description"] == "") {
		return fail("a skill needs a name and a description in its frontmatter")
	}
	if help.Description == "" {
		help.Description = firstBodyLine(body)
	}
	if help.Description == "" {
		return fail("renders no help: no description and no body text")
	}
	return help
}

// pageFrontmatter splits a page that opens a frontmatter block into the
// block's interior lines, line endings removed, and the body after it. The
// block closes at the first column-0 `---` (frontmatter.IsDelimiter, the one
// delimiter rule) or at YAML's document-end marker `...`, which a YAML reader
// accepts in its place. closed is false when neither closes it.
func pageFrontmatter(text string) (inner []string, body string, closed bool) {
	lines := strings.SplitAfter(text, "\n")
	n := len(lines[0])
	for i := 1; i < len(lines); i++ {
		ln := lines[i]
		column0 := !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "\t")
		if column0 && (frontmatter.IsDelimiter(ln) || strings.TrimRight(ln, " \t\r\n") == "...") {
			for _, l := range lines[1:i] {
				inner = append(inner, strings.TrimRight(l, "\r\n"))
			}
			return inner, text[n+len(ln):], true
		}
		n += len(ln)
	}
	return nil, text, false
}

// How a key's value is spelled, which decides what its indented lines are.
const (
	valueScalar     = iota // a same-line scalar, which indented lines continue
	valueBlock             // a block scalar (`>` or `|`): the indented lines are its text
	valueOpen              // nothing on the key's line: the first indented line decides
	valueCollection        // a nested mapping or sequence, which help never reads
)

// pageFields reads a closed frontmatter block's top-level mapping, refusing a
// column-0 line no YAML mapping holds and a duplicated key. A scalar continued
// on indented lines (plain or quoted) is folded into one line, and so is a
// block scalar: one line is all help needs. inner is the block's interior, so
// its first line is the file's second.
func pageFields(inner []string) (map[string]string, string) {
	fields := map[string]string{}
	seen := map[string]int{}
	var key string
	kind := valueCollection
	var parts []string
	flush := func() {
		switch {
		case key == "":
		case kind == valueScalar:
			v := strings.TrimSpace(frontmatter.StripComment(strings.Join(parts, " ")))
			if scalar, ok := frontmatter.ScalarString(v); ok {
				v = scalar
			}
			fields[key] = v
		case kind == valueBlock:
			fields[key] = strings.Join(parts, " ")
		default:
			fields[key] = ""
		}
		key, kind, parts = "", valueCollection, nil
	}
	for i, raw := range inner {
		lineNo := i + 2
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			switch {
			case kind == valueBlock:
				parts = append(parts, trimmed)
			case strings.HasPrefix(trimmed, "#"):
				// A comment line, inside no block scalar.
			case kind == valueOpen && opensCollection(trimmed):
				kind = valueCollection
			case kind == valueOpen:
				kind, parts = valueScalar, []string{trimmed}
			case kind == valueScalar:
				parts = append(parts, trimmed)
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if line == "-" || strings.HasPrefix(line, "- ") {
			// A block sequence at the key's own indentation is legal YAML.
			if kind == valueOpen {
				kind = valueCollection
			}
			continue
		}
		flush()
		k, rest, ok := mappingKey(line)
		if !ok {
			return nil, fmt.Sprintf("frontmatter line %d is not a YAML mapping entry: %q", lineNo, trimmed)
		}
		if first, dup := seen[k]; dup {
			return nil, fmt.Sprintf("frontmatter key %q is a duplicate (lines %d and %d)", k, first, lineNo)
		}
		seen[k] = lineNo
		key = k
		switch value := strings.TrimSpace(frontmatter.StripComment(rest)); {
		case value == "":
			kind = valueOpen
		case frontmatter.BlockScalarHeaderRe.MatchString(value):
			kind = valueBlock
		default:
			kind, parts = valueScalar, []string{strings.TrimSpace(rest)}
		}
	}
	flush()
	return fields, ""
}

// opensCollection reports whether the first indented line under a key with no
// same-line value starts a nested collection rather than a plain scalar: a
// sequence entry, or a mapping entry.
func opensCollection(trimmed string) bool {
	if trimmed == "-" || strings.HasPrefix(trimmed, "- ") {
		return true
	}
	_, _, ok := mappingKey(trimmed)
	return ok
}

// plainKeyIndicators are the characters a plain (unquoted) YAML key cannot
// start with; `-`, `?` and `:` join them only when a space follows.
const plainKeyIndicators = "[]{},#&*!|>'\"%@`"

// mappingKey reads a column-0 block mapping entry: a plain, double-quoted or
// single-quoted key, then a colon that a space, a tab or the end of the line
// follows. rest is what follows the colon. A plain key may hold any character
// YAML allows there, non-ASCII included.
func mappingKey(line string) (key, rest string, ok bool) {
	if line == "" {
		return "", "", false
	}
	colonAt := func(after string) (string, bool) {
		after = strings.TrimLeft(after, " \t")
		if !strings.HasPrefix(after, ":") {
			return "", false
		}
		if tail := after[1:]; tail == "" || tail[0] == ' ' || tail[0] == '\t' {
			return tail, true
		}
		return "", false
	}
	switch line[0] {
	case '"':
		for i := 1; i < len(line); i++ {
			switch line[i] {
			case '\\':
				i++
			case '"':
				r, ok := colonAt(line[i+1:])
				return frontmatter.Unquote(line[1:i]), r, ok
			}
		}
		return "", "", false
	case '\'':
		for i := 1; i < len(line); i++ {
			if line[i] != '\'' {
				continue
			}
			if i+1 < len(line) && line[i+1] == '\'' {
				i++
				continue
			}
			r, ok := colonAt(line[i+1:])
			return strings.ReplaceAll(line[1:i], "''", "'"), r, ok
		}
		return "", "", false
	}
	if strings.ContainsRune(plainKeyIndicators, rune(line[0])) {
		return "", "", false
	}
	if strings.ContainsRune("-?:", rune(line[0])) && (len(line) == 1 || line[1] == ' ' || line[1] == '\t') {
		return "", "", false
	}
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '#':
			if i > 0 && (line[i-1] == ' ' || line[i-1] == '\t') {
				return "", "", false // a comment starts before any colon
			}
		case ':':
			if i+1 == len(line) || line[i+1] == ' ' || line[i+1] == '\t' {
				k := strings.TrimRight(line[:i], " \t")
				return k, line[i+1:], k != ""
			}
		}
	}
	return "", "", false
}

// firstBodyLine is the first non-blank line of a page's body.
func firstBodyLine(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

// deepPages is the list of pages the deep tier renders: every command, skill
// and agent page the resolved surface declares and the payload carries.
func deepPages(tree PayloadTree, surface InstallSurface) []PageRef {
	seen := map[string]bool{}
	var pages []PageRef
	for _, e := range surface.Entries {
		switch e.Kind {
		case SurfaceCommand, SurfaceSkill, SurfaceAgent:
		default:
			continue
		}
		if e.Requirement != RequirePayload || !strings.HasSuffix(e.Path, ".md") || seen[e.Path] || !tree.Has(e.Path) {
			continue
		}
		seen[e.Path] = true
		pages = append(pages, PageRef{Kind: e.Kind, Path: e.Path})
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Path < pages[j].Path })
	return pages
}

// SmokeDeep runs the deep tier over a materialised payload at root, resolving
// the SAME declared surface the light tier asserts over and handing every page
// to run. Like SmokeLight it never returns an error: a runner that fails is the
// most serious finding it can make.
func SmokeDeep(root string, run PageRunner) DeepSmokeReport {
	rep := DeepSmokeReport{Tier: SmokeTierDeep}
	tree := NewDirTree(root)
	surface, err := ResolveInstallSurface(tree)
	if err != nil {
		rep.Findings = append(rep.Findings, SmokeFinding{Kind: findingManifestUnreadable, Detail: err.Error()})
		return rep
	}
	pages := deepPages(tree, surface)
	rep.Checked = len(pages)
	if run == nil {
		rep.Findings = append(rep.Findings, SmokeFinding{Kind: findingDeepSmokeUnavailable,
			Detail: "the deep tier has no isolated runner, so no page was rendered"})
		return rep
	}
	answers, err := run(root, pages)
	if err != nil {
		rep.Findings = append(rep.Findings, SmokeFinding{Kind: findingDeepSmokeUnavailable,
			Detail: "the isolated page runner failed, so no page is known to load: " + err.Error()})
		return rep
	}
	byPath := make(map[string]PageHelp, len(answers))
	for _, a := range answers {
		byPath[a.Path] = a
	}
	for _, p := range pages {
		a, ok := byPath[p.Path]
		if !ok {
			rep.Findings = append(rep.Findings, SmokeFinding{Kind: findingDeepSmokeUnavailable, Path: p.Path,
				Detail: fmt.Sprintf("the isolated page runner returned no answer for %s %q", p.Kind, p.Path)})
			continue
		}
		rep.Pages = append(rep.Pages, a)
		if a.Error != "" {
			rep.Findings = append(rep.Findings, SmokeFinding{Kind: findingPageUnloadable, Path: p.Path,
				Detail: fmt.Sprintf("%s %q does not load: %s", p.Kind, p.Path, a.Error)})
		}
	}
	rep.OK = len(rep.Findings) == 0
	return rep
}

// smokeDeepOverBundle materialises the resolved bundle into a private
// temporary tree, runs the deep tier rooted there, and removes the tree. The
// materialisation is the bundle's bytes, unstamped: a version stamp adds keys
// to the two manifests and changes no page.
func smokeDeepOverBundle(bundle Bundle, run PageRunner) DeepSmokeReport {
	tmp, err := os.MkdirTemp("", "abcd-deep-smoke-")
	if err != nil {
		return deepUnavailable(err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	dir := filepath.Join(tmp, "payload")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return deepUnavailable(err)
	}
	for _, f := range bundle.Included {
		if err := copyPayloadFile(dir, f); err != nil {
			return deepUnavailable(fmt.Errorf("materialising %s: %w", f.LogicalPath, pathFreeError(err)))
		}
	}
	return SmokeDeep(dir, run)
}

func deepUnavailable(err error) DeepSmokeReport {
	return DeepSmokeReport{Tier: SmokeTierDeep, Findings: []SmokeFinding{{
		Kind: findingDeepSmokeUnavailable, Detail: "the payload could not be materialised for the deep tier: " + pathFreeError(err).Error(),
	}}}
}

// ErrPayloadPageUnloadable is the precheck's refusal when a declared page
// would not load. It wraps ErrPayloadUninstallable, so a caller matching the
// installability refusal matches this one too.
var ErrPayloadPageUnloadable = fmt.Errorf("%w (deep tier)", ErrPayloadUninstallable)

// deepSmokeRefusals renders a failed deep tier as the refusal lines a gate
// reports, each naming its page.
func deepSmokeRefusals(rep *DeepSmokeReport) []string {
	if rep == nil {
		return nil
	}
	var out []string
	for _, f := range rep.Findings {
		out = append(out, "installability smoke (deep): "+f.Detail)
	}
	return out
}

// deepSmokeGate is the deep tier's gate row.
func deepSmokeGate(rep *DeepSmokeReport) GateSummary {
	return GateSummary{
		Name: "installability-smoke-deep", Status: "ran", Tier: TierHardFail,
		Detail: "rendered " + itoa(rep.Checked) + " page(s) in an isolated subprocess, " + itoa(len(rep.Findings)) + " findings",
	}
}
