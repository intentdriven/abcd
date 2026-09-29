// Package positioning holds a repository's canonical self-description — its
// title, tagline, and elevator pitch — and the deterministic check that every
// rendered surface still says it.
//
// The canonical home is a markdown block in the repo's own record (markdown
// stays the single source of truth); the committed configuration records only
// where that block lives and which surfaces render from it. The check is
// advisory by default and never rewrites a surface: a re-render is always a
// proposed diff a human adopts (spc-19, itd-102).
//
// This is repo POSITIONING, not commit-author identity: the git author gate
// pinned in .abcd/config/identity.json lives in internal/core/identity and
// shares nothing with this package but the English word.
package positioning

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// maxBlockFileBytes caps a guarded read of the block's host file. The block
// lives in a committed record file; a megabyte is far past any real one and
// stops a device or runaway file being read unbounded.
const maxBlockFileBytes = 1 << 20

// Parse failures, as sentinels so a caller can distinguish "the repo has not
// adopted a block" from "the block is malformed" without string matching.
var (
	// ErrBlockFileMissing: the configured file does not exist (or is not a
	// regular file).
	ErrBlockFileMissing = errors.New("identity block file not found")
	// ErrHeadingMissing: the file exists but carries no such heading.
	ErrHeadingMissing = errors.New("identity block heading not found")
	// ErrBulletMissing: the heading exists but a required bullet (title or
	// tagline) is absent or empty.
	ErrBulletMissing = errors.New("identity block is missing a required bullet")
	// ErrBadLocation: the configured location is not a safe repo-relative path.
	ErrBadLocation = errors.New("identity block location is not a repo-relative path")
)

// BlockLocation is where the identity block lives: a repo-relative markdown
// file and the heading the block sits under. It is committed configuration, not
// derived, so both fields are validated before use.
type BlockLocation struct {
	File    string `json:"file"`
	Heading string `json:"heading"`
}

// Block is a repository's canonical self-description. Title and Tagline are
// required; Pitch is optional at onboarding. File and Line cite where it was
// parsed from so a finding can point back at the canon.
type Block struct {
	Title   string `json:"title"`
	Tagline string `json:"tagline"`
	Pitch   string `json:"pitch,omitempty"`
	File    string `json:"file"`
	Line    int    `json:"line"`
}

// Field returns a block field by its lower-case name (the vocabulary a surface's
// requires/template use), reporting whether the name is one of the three.
func (b Block) Field(name string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "title":
		return b.Title, true
	case "tagline":
		return b.Tagline, true
	case "pitch":
		return b.Pitch, true
	}
	return "", false
}

// headingRe matches an ATX heading and captures its level and text.
var headingRe = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*\s*$`)

// bulletRe matches the fixed identity bullet shape — `- **Label:** value` (a `-`
// or `*` marker, up to three leading spaces) — capturing the label and the
// value. The value may be empty when the bullet wraps onto continuation lines.
var bulletRe = regexp.MustCompile(`^ {0,3}[-*]\s+\*\*([A-Za-z]+):\*\*\s*(.*)$`)

// ParseBlock reads the identity block at loc under root. The heading match is
// case-insensitive and level-agnostic; the block runs to the next heading of the
// same or shallower level (or end of file). A bullet's value may wrap onto
// indented continuation lines, which are joined with a single space.
//
// Title and Tagline are required; Pitch is optional. Every failure mode is a
// sentinel-wrapped error, never a zero Block with a nil error — a caller must
// not read "no block" as "the block says nothing".
func ParseBlock(root string, loc BlockLocation) (Block, error) {
	if !validBlockFile(loc.File) {
		return Block{}, fmt.Errorf("%w: %q", ErrBadLocation, loc.File)
	}
	r, err := openRepoRoot(root)
	if err != nil {
		return Block{}, fmt.Errorf("%w: %s: %s", ErrBlockFileMissing, loc.File, pathFreeReason(err))
	}
	defer r.Close()
	return parseBlockIn(r, loc)
}

// validBlockFile is the one gate on a block location's file, shared by every
// entry that reads the block (ParseBlock, parseBlockIn) or writes it (Init). A
// clean repo-relative path is not enough: ".git/config" is one, and the block is
// read and rendered as identity output — or, through Init, written into — so the
// git directory is refused here as the registry's surfaces refuse it
// (fsutil.InsideGitDir, iss-2608291814578333).
func validBlockFile(p string) bool {
	return fsutil.ValidRelPath(p) && !fsutil.InsideGitDir(p)
}

// openRepoRoot opens root as an os.Root containment scope. Every positioning
// read and write resolves through one of these, so a repository that COMMITS a
// symlinked directory (git mode 120000) as an ancestor of a configured path
// cannot walk the operation outside itself: fsutil.ValidRelPath is lexical and
// cannot see a symlink, and a leaf-only O_NOFOLLOW guard never looks at the
// ancestors. The OS enforces containment instead.
//
// A root is opened per operation and closed by the caller, never cached: a
// cached handle would answer for a tree that has since moved.
func openRepoRoot(root string) (*os.Root, error) {
	return os.OpenRoot(root)
}

// parseBlockIn is ParseBlock with the containment root already open, so a caller
// that is already reading the repo (Check, Init) opens one root for the whole
// operation.
func parseBlockIn(r *os.Root, loc BlockLocation) (Block, error) {
	if !validBlockFile(loc.File) {
		return Block{}, fmt.Errorf("%w: %q", ErrBadLocation, loc.File)
	}
	heading := strings.TrimSpace(loc.Heading)
	if heading == "" {
		return Block{}, fmt.Errorf("%w: heading is empty", ErrBadLocation)
	}

	data, err := fsutil.ReadGuardedInRoot(r, loc.File, maxBlockFileBytes)
	if err != nil {
		// A missing, symlinked, oversize, or root-escaping block file is all one
		// thing to a caller — the canon could not be read where the configuration
		// said it was. The reason rides along, but stripped of its absolute path: this
		// error reaches an audit finding, which must never carry one (iss-81).
		return Block{}, fmt.Errorf("%w: %s: %s", ErrBlockFileMissing, loc.File, pathFreeReason(err))
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	start, level := findHeading(lines, heading)
	if start < 0 {
		return Block{}, fmt.Errorf("%w: %q in %s", ErrHeadingMissing, heading, loc.File)
	}

	b := Block{File: loc.File, Line: start + 1}
	for _, bl := range collectBullets(lines, start+1, level) {
		switch strings.ToLower(bl.label) {
		case "title":
			b.Title = bl.value
		case "tagline":
			b.Tagline = bl.value
		case "pitch":
			b.Pitch = bl.value
		}
	}
	for _, req := range []struct{ name, value string }{{"Title", b.Title}, {"Tagline", b.Tagline}} {
		if req.value == "" {
			return Block{}, fmt.Errorf("%w: **%s:** under %q in %s", ErrBulletMissing, req.name, heading, loc.File)
		}
	}
	return b, nil
}

// pathFreeReason renders a filesystem error without the absolute path it was
// raised against. Positioning errors surface in audit findings and machine
// output, which must never carry a developer-identity path (iss-81).
func pathFreeReason(err error) string {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Op + ": " + pe.Err.Error()
	}
	return err.Error()
}

// findHeading returns the index of the first heading whose text equals want
// (case-insensitively, ignoring surrounding space) and its level, or -1.
func findHeading(lines []string, want string) (idx, level int) {
	want = strings.ToLower(want)
	for i, line := range lines {
		m := headingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if strings.ToLower(strings.TrimSpace(m[2])) == want {
			return i, len(m[1])
		}
	}
	return -1, 0
}

type bullet struct{ label, value string }

// collectBullets gathers the identity bullets between the heading at from-1 and
// the next heading of level <= level (or end of file), folding each bullet's
// indented continuation lines into its value.
func collectBullets(lines []string, from, level int) []bullet {
	var out []bullet
	for i := from; i < len(lines); i++ {
		line := lines[i]
		if m := headingRe.FindStringSubmatch(line); m != nil && len(m[1]) <= level {
			break
		}
		m := bulletRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		parts := []string{strings.TrimSpace(m[2])}
		// Continuations: indented, non-blank lines that are not themselves a
		// list item or a heading. A blank line ends the bullet.
		for j := i + 1; j < len(lines); j++ {
			next := lines[j]
			if strings.TrimSpace(next) == "" || !isIndentedContinuation(next) {
				break
			}
			parts = append(parts, strings.TrimSpace(next))
			i = j
		}
		out = append(out, bullet{label: m[1], value: strings.TrimSpace(strings.Join(nonEmpty(parts), " "))})
	}
	return out
}

// isIndentedContinuation reports whether line continues the preceding bullet:
// it starts with whitespace and does not open a new list item or heading.
func isIndentedContinuation(line string) bool {
	if line == "" || !(line[0] == ' ' || line[0] == '\t') {
		return false
	}
	t := strings.TrimSpace(line)
	if strings.HasPrefix(t, "#") {
		return false
	}
	if len(t) >= 2 && (t[0] == '-' || t[0] == '*' || t[0] == '+') && (t[1] == ' ' || t[1] == '\t') {
		return false
	}
	return true
}

// nonEmpty drops empty strings so a label-only bullet with a wrapped value does
// not gain a leading space.
func nonEmpty(in []string) []string {
	out := in[:0:0]
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
