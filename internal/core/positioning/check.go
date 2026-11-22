package positioning

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"go.yaml.in/yaml/v3"
)

// maxSurfaceBytes caps a guarded surface read. A README or manifest is small;
// the cap stops a device or runaway file being read unbounded.
const maxSurfaceBytes = 1 << 20

// ErrSurfaceUnreadable: a registered surface file is present but could not be
// read safely — it escapes the repository root through a symlinked component,
// its leaf is a symlink or a device, or it is oversize. It is never an absence:
// a surface that cannot be read must not read as "no drift".
var ErrSurfaceUnreadable = errors.New("surface file could not be read")

// Status is one surface's outcome.
type Status string

const (
	// StatusOK: the surface carries every block field it declares.
	StatusOK Status = "ok"
	// StatusDrifted: the surface was located but no longer says what the block
	// says.
	StatusDrifted Status = "drifted"
	// StatusAbsent: none of the surface's candidate files exists, so the
	// surface is not present in this repo. Not a finding.
	StatusAbsent Status = "absent"
	// StatusUnlocatable: the file exists but the locator found nothing. A
	// finding: a locator that stopped matching hides drift rather than proving
	// its absence.
	StatusUnlocatable Status = "unlocatable"
)

// SurfaceResult is one surface's comparison against the block.
type SurfaceResult struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
	// File is the candidate that was checked (empty when all were absent).
	File string `json:"file,omitempty"`
	// Line is the 1-based line the located text starts on.
	Line int `json:"line,omitempty"`
	// Found is the located text, verbatim — the exact drifted content.
	Found string `json:"found,omitempty"`
	// Missing names the block fields the surface no longer carries.
	Missing []string `json:"missing,omitempty"`
	// Canonical is what the block says this surface should carry.
	Canonical string `json:"canonical,omitempty"`
}

// Report is a whole positioning check: the block it was held to, the family
// severity, and one result per registered surface.
type Report struct {
	Block    Block           `json:"block"`
	Severity string          `json:"severity"`
	Surfaces []SurfaceResult `json:"surfaces"`
}

// Drifted counts the surfaces that need attention — drifted or unlocatable.
func (r Report) Drifted() int {
	n := 0
	for _, s := range r.Surfaces {
		if s.Status == StatusDrifted || s.Status == StatusUnlocatable {
			n++
		}
	}
	return n
}

// Check parses the identity block and compares every registered surface against
// it. It reads only; nothing is written and nothing is proposed here. An
// unreadable or malformed block is an error, never an empty clean report — a
// check that cannot run must not read as a pass.
func Check(root string, cfg Config) (Report, error) {
	if err := cfg.Validate(); err != nil {
		return Report{}, err
	}
	r, err := openRepoRoot(root)
	if err != nil {
		return Report{}, fmt.Errorf("%w: %s: %s", ErrBlockFileMissing, cfg.Block.File, pathFreeReason(err))
	}
	defer r.Close()
	block, err := parseBlockIn(r, cfg.Block)
	if err != nil {
		return Report{}, err
	}
	rep := Report{Block: block, Severity: cfg.severity()}
	for _, s := range cfg.surfaces() {
		res, err := checkSurface(r, s, block)
		if err != nil {
			return Report{}, err
		}
		rep.Surfaces = append(rep.Surfaces, res)
	}
	return rep, nil
}

// checkSurface resolves the surface's file, locates its self-description, and
// compares it to the block.
func checkSurface(r *os.Root, s Surface, b Block) (SurfaceResult, error) {
	res := SurfaceResult{ID: s.ID, Status: StatusAbsent, Canonical: s.render(b)}

	rel, data, ok, err := readFirstPresent(r, s.Files)
	if err != nil {
		return SurfaceResult{}, err
	}
	if !ok {
		return res, nil
	}
	res.File = rel

	found, line, located, err := s.locate(data)
	if err != nil {
		return SurfaceResult{}, fmt.Errorf("surface %q in %s: %w", s.ID, rel, err)
	}
	if !located {
		res.Status = StatusUnlocatable
		return res, nil
	}
	res.Found, res.Line = found, line

	hay := normalize(found)
	for _, field := range s.Requires {
		want, _ := b.Field(field)
		if want == "" {
			// An optional field the block does not carry cannot be required of
			// a surface; skip it rather than fabricating drift.
			continue
		}
		if !strings.Contains(hay, needle(want)) {
			res.Missing = append(res.Missing, field)
		}
	}
	res.Status = StatusOK
	if len(res.Missing) > 0 {
		res.Status = StatusDrifted
	}
	return res, nil
}

// readFirstPresent returns the first candidate that exists, its repo-relative
// path, and its contents. Every read is guarded AND contained: the file names
// come from committed configuration in an arbitrary audited repo, so a candidate
// may be a symlink, a device, enormous, or sitting behind a symlinked ancestor
// directory that reaches outside the repository entirely. ValidRelPath is the
// cheap lexical reject; the containment root is what actually holds the read
// inside the repo, and a surface's content is quoted verbatim into the report,
// so nothing from outside may ever be read here.
func readFirstPresent(r *os.Root, files []string) (rel string, data []byte, ok bool, err error) {
	for _, f := range files {
		if !fsutil.ValidRelPath(f) {
			return "", nil, false, fmt.Errorf("%w: surface file %q is not a repo-relative path", ErrConfigInvalid, f)
		}
		b, rerr := fsutil.ReadGuardedInRoot(r, f, maxSurfaceBytes)
		if rerr != nil {
			if os.IsNotExist(rerr) {
				continue
			}
			// A present-but-unreadable surface (symlinked, oversize, a device, a
			// path escaping the root) is not an absence: report it rather than
			// skipping it silently. Path-free — this reaches an audit finding
			// (iss-81).
			return f, nil, false, fmt.Errorf("%w: %s: %s", ErrSurfaceUnreadable, f, pathFreeReason(rerr))
		}
		return f, b, true, nil
	}
	return "", nil, false, nil
}

// span is the byte range of a surface's located self-description within its
// file, plus the decoded text. Keeping the range (not just the text) is what
// lets the re-render propose a replacement in place rather than guessing where
// the text sat.
type span struct {
	start, end int    // byte offsets into the normalised file text
	text       string // the decoded value (JSON-unescaped for a manifest field)
	// escaped reports that the byte range holds an escaped form, so a
	// replacement must be re-escaped before it is spliced back in.
	escaped bool
}

// locate extracts the surface's self-description and the 1-based line it starts
// on, reporting whether the locator matched at all.
func (s Surface) locate(data []byte) (found string, line int, ok bool, err error) {
	text := normalizeNewlines(data)
	sp, ok, err := s.locateSpan(text)
	if err != nil || !ok {
		return "", 0, false, err
	}
	if sp.start < 0 {
		// Located by value but not pinned to a byte range: no citable line.
		return sp.text, 0, true, nil
	}
	return sp.text, 1 + strings.Count(text[:sp.start], "\n"), true, nil
}

// locateSpan finds the byte range the surface's self-description occupies.
func (s Surface) locateSpan(text string) (span, bool, error) {
	switch s.Kind {
	case KindJSONField:
		return locateJSONField(text, s.Field)
	case KindRegexp:
		for _, p := range s.Patterns {
			re, cerr := regexp.Compile(p)
			if cerr != nil {
				return span{}, false, fmt.Errorf("pattern %q does not compile: %w", p, cerr)
			}
			loc := re.FindStringSubmatchIndex(text)
			if loc == nil || loc[2] < 0 {
				continue
			}
			// A pattern whose paragraph terminator is end-of-file captures the
			// file's final newline(s). They are not part of what the surface
			// SAYS, and a replacement span that swallowed them would delete the
			// file's line ending, so the span stops at the last real character.
			start, end := loc[2], loc[3]
			for end > start && (text[end-1] == '\n' || text[end-1] == '\r') {
				end--
			}
			sp := span{start: start, end: end, text: text[start:end]}
			if s.Escape != "" {
				// The capture is the body of a quoted string: compare what it
				// says, not how it is spelled, and have a proposal re-escape.
				// A body that does not decode is compared as written, and the
				// proposal that replaces it is escaped correctly.
				sp.escaped = true
				if v, ok := decodeQuoted(s.Escape, sp.text); ok {
					sp.text = v
				}
			}
			return sp, true, nil
		}
		return span{}, false, nil
	}
	return span{}, false, fmt.Errorf("%w: unknown surface kind %q", ErrConfigInvalid, s.Kind)
}

// jsonStringMemberFmt builds a regexp for a JSON object member whose value is a
// string, capturing the escaped value between the quotes. It recovers the byte
// range the JSON decoder never reports; the authoritative parse decides whether
// that range really is the top-level field asked for.
const jsonStringMemberFmt = `"%s"\s*:\s*"((?:[^"\\]|\\.)*)"`

// locateJSONField reads a top-level string field of a JSON manifest and finds
// the byte range its value occupies. The value is taken from a real JSON parse
// (so a nested key of the same name can never be mistaken for the top-level
// one); the byte range comes from a scan, and is accepted only when it decodes
// to exactly the parsed value.
func locateJSONField(text, field string) (span, bool, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		// A manifest that is not a JSON object cannot be located; that is a
		// locator miss, not a fault that should abort the whole audit.
		return span{}, false, nil
	}
	raw, present := obj[field]
	if !present {
		return span{}, false, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return span{}, false, nil // present but not a string
	}
	re, err := regexp.Compile(fmt.Sprintf(jsonStringMemberFmt, regexp.QuoteMeta(field)))
	if err != nil {
		return span{}, false, nil
	}
	for _, loc := range re.FindAllStringSubmatchIndex(text, -1) {
		var decoded string
		if json.Unmarshal([]byte(`"`+text[loc[2]:loc[3]]+`"`), &decoded) != nil || decoded != value {
			continue
		}
		return span{start: loc[2], end: loc[3], text: value, escaped: true}, true, nil
	}
	// The value parsed but its bytes could not be pinned (an unusual encoding).
	// Report the value with no usable range: the check still runs, the
	// re-render declines to propose rather than splicing at a guessed offset.
	return span{start: -1, end: -1, text: value}, true, nil
}

// decodeQuoted reads body as the inside of a quoted string in the named
// format, reporting whether it decodes.
func decodeQuoted(format, body string) (string, bool) {
	var v string
	switch format {
	case EscapeJSON:
		if json.Unmarshal([]byte(`"`+body+`"`), &v) != nil {
			return "", false
		}
	case EscapeYAML:
		if yaml.Unmarshal([]byte(`"`+body+`"`), &v) != nil {
			return "", false
		}
	default:
		return "", false
	}
	return v, true
}

// normalizeNewlines renders file bytes as the text every locator works over.
func normalizeNewlines(data []byte) string {
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

var (
	// emphasisRe strips inline markdown emphasis and code markers, so a tagline
	// bolded inside a sentence still reads as the tagline.
	emphasisRe = regexp.MustCompile("[*_`]+")
	// htmlTagRe strips inline HTML tags, so an anchor or span wrapped around
	// part of a strapline does not count as drift.
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)
	// spaceRe collapses every whitespace run, so line wrapping is invisible to
	// the comparison.
	spaceRe = regexp.MustCompile(`\s+`)
	// dashRe folds the dash family and non-breaking spaces onto their plain
	// forms, so typographic polish is not drift.
	dashRe = strings.NewReplacer(
		"—", "-", "–", "-", "‒", "-", "‑", "-", "−", "-",
		" ", " ", "‘", "'", "’", "'", "“", `"`, "”", `"`,
	)
)

// normalize reduces text to the form the comparison is made in: HTML and
// markdown markup stripped, dashes and quotes folded, whitespace collapsed,
// lower-cased. It is deliberately lossy — the check asks whether a surface still
// SAYS the canonical line, not whether it renders it byte for byte.
func normalize(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = emphasisRe.ReplaceAllString(s, "")
	s = dashRe.Replace(s)
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.ToLower(strings.TrimSpace(s))
}

// needle is a block field normalised for containment: terminal sentence
// punctuation is dropped so a tagline that ends a sentence in one surface and
// continues into a clause in another still matches.
func needle(field string) string {
	return strings.TrimRight(normalize(field), " .!?;:,")
}
