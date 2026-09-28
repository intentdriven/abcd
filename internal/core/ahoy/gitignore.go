package ahoy

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Canonical .gitignore fence markers and the do-not-hand-edit header.
const (
	gitignoreBegin  = "# BEGIN ABCD"
	gitignoreEnd    = "# END ABCD"
	gitignoreHeader = "# abcd-managed block — do not hand-edit. Run /abcd:ahoy to refresh."
)

// gitignoreMaxBytes caps how large a .gitignore we will round-trip.
const gitignoreMaxBytes = 256 * 1024

// visibilityEntries is the canonical abcd-managed entry set per visibility,
// per the brief's visibility table (§1). Order preserved for stable diffs.
// A private repo commits the whole .abcd/ namespace and gitignores only the
// local-ephemeral tier; a public repo gitignores that namespace outright — one
// switch, no per-subdirectory exceptions, so .abcd/.work.local/ needs no
// separate entry there — plus the legacy root-level memory/ snapshot.
// The table means the repo-root directories, so the public entries are
// anchored: a gitignore pattern with no non-trailing slash matches at any
// depth, and unanchored .abcd/ or memory/ would also ignore e.g. a nested
// src/.abcd/ or an internal/memory/ source package. The leading slash anchors
// the pattern to the .gitignore's own directory — the repository root. The
// private entry's internal slash already anchors it, so it takes no leading
// slash.
var visibilityEntries = map[string][]string{
	"private": {".abcd/.work.local/"},
	"public":  {"/.abcd/", "/memory/"},
}

// effectiveVisibilityEntries resolves the entry set actually written for a
// repo, and whether it was narrowed. The table's public set fences the whole
// .abcd/ namespace — but an ignore rule cannot untrack committed files, so on
// a repo that COMMITS the record tiers (the documented three-tier layout, and
// this repository itself) the wholesale fence only hides every NEW record from
// git status and makes `git add` refuse them (iss-255). When .abcd/ holds
// tracked files, the /.abcd/ entry — and ONLY that entry — narrows to the
// local-ephemeral tier; every other declared entry (the /memory/ snapshot
// fence) is kept, because its justification is untouched by the tracked-tier
// argument. Narrowing needs positive evidence: a repo whose git cannot be
// asked keeps the declared set — the fence is the public setting's whole
// purpose, and a repair applied on no evidence would remove it. A directory
// that is not a git repository at all likewise keeps the declared set: no git
// means no ignore semantics to break.
func effectiveVisibilityEntries(cwd, visibility string) (entries []string, narrowed bool) {
	declared, ok := visibilityEntries[visibility]
	if !ok || visibility != "public" {
		return declared, false
	}
	if !gitutil.InRepo(cwd) {
		return declared, false
	}
	out, err := gitutil.RunLimited(cwd, 4096, "ls-files", "--", ".abcd")
	if err != nil || strings.TrimSpace(out) == "" {
		return declared, false
	}
	narrowedSet := make([]string, 0, len(declared))
	for _, e := range declared {
		if e == "/.abcd/" {
			narrowedSet = append(narrowedSet, visibilityEntries["private"]...)
			continue
		}
		narrowedSet = append(narrowedSet, e)
	}
	return narrowedSet, true
}

// canonicalGitignoreBlock returns the block lines (EOL-naive) for an entry set.
func canonicalGitignoreBlock(entries []string) []string {
	lines := []string{gitignoreBegin, gitignoreHeader}
	lines = append(lines, entries...)
	lines = append(lines, gitignoreEnd)
	return lines
}

// gitignoreEOL returns "\r\n" when the first newline is CRLF, else "\n".
func gitignoreEOL(raw []byte) string {
	nl := bytes.IndexByte(raw, '\n')
	if nl == -1 {
		return "\n"
	}
	if nl > 0 && raw[nl-1] == '\r' {
		return "\r\n"
	}
	return "\n"
}

// gitignoreBlockDrifts reports whether the abcd-managed block in cwd/.gitignore
// differs from the canonical entry set for visibility. Read-only; fail-closed
// (drift) on any unsafe/unreadable shape so apply is offered.
func gitignoreBlockDrifts(cwd, visibility string) bool {
	if _, ok := visibilityEntries[visibility]; !ok {
		return false
	}
	entries, _ := effectiveVisibilityEntries(cwd, visibility)
	path := filepath.Join(cwd, ".gitignore")
	// ReadGuarded folds the absent / symlink / non-regular / oversize / unreadable
	// cases into one guarded open (iss-109): every one is drift here (fail-closed),
	// so no distinct signal is lost by collapsing the manual Lstat guard into it.
	raw, err := fsutil.ReadGuarded(path, gitignoreMaxBytes)
	if err != nil {
		return true
	}
	inner, count := extractGitignoreBlock(raw)
	if count == 0 {
		return true // no abcd block — apply plants it
	}
	if count > 1 {
		return true // duplicate blocks — collapse on apply
	}
	current := parseGitignoreEntries(inner)
	expected := make(map[string]bool, len(entries))
	for _, e := range entries {
		expected[e] = true
	}
	if len(current) != len(expected) {
		return true
	}
	for e := range current {
		if !expected[e] {
			return true
		}
	}
	return false
}

// extractGitignoreBlock returns the inner bytes between the single BEGIN/END
// pair and the number of pairs found. count==0 => no block; count>1 => drift.
func extractGitignoreBlock(raw []byte) (inner []byte, count int) {
	begin := []byte(gitignoreBegin)
	end := []byte(gitignoreEnd)
	beginCount := bytes.Count(raw, begin)
	endCount := bytes.Count(raw, end)
	if beginCount == 0 && endCount == 0 {
		return nil, 0
	}
	if beginCount != 1 || endCount != 1 {
		return nil, 2
	}
	bi := bytes.Index(raw, begin)
	lineEnd := bytes.IndexByte(raw[bi:], '\n')
	if lineEnd == -1 {
		return nil, 2
	}
	lineEnd += bi
	ei := bytes.Index(raw[lineEnd+1:], end)
	if ei == -1 {
		return nil, 2
	}
	ei += lineEnd + 1
	return raw[lineEnd+1 : ei], 1
}

// parseGitignoreEntries returns the entry set inside a block body, stripping
// comments and blank lines and preserving a trailing slash.
func parseGitignoreEntries(block []byte) map[string]bool {
	entries := map[string]bool{}
	for _, line := range strings.Split(string(block), "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		entries[s] = true
	}
	return entries
}

// applyVisibilityBlock ensures cwd/.gitignore carries exactly one canonical
// abcd block for visibility. It strips every existing block (including
// unbalanced fragments and stray END lines) then appends one canonical block,
// preserving non-block content. Returns (wrote, err). Fail-closed on unsafe
// files: an error is returned and the user's file is preserved.
func applyVisibilityBlock(cwd, visibility string) (bool, error) {
	if _, ok := visibilityEntries[visibility]; !ok {
		return false, &ahoyError{"unknown visibility: " + visibility}
	}
	entries, _ := effectiveVisibilityEntries(cwd, visibility)
	path := filepath.Join(cwd, ".gitignore")
	var wrote bool
	err := withRewriteLock(path, func() error {
		var err error
		wrote, err = rewriteVisibilityBlock(path, entries)
		return err
	})
	return wrote, err
}

// rewriteVisibilityBlock is applyVisibilityBlock's read, change and write of
// path, run under the file's lock (withRewriteLock) so an edit another abcd
// makes to .gitignore between the read and the write is kept (iss-127).
func rewriteVisibilityBlock(path string, entries []string) (bool, error) {
	// Keep the symlink pre-check: it yields a DISTINCT refusal that ReadGuarded's
	// O_NOFOLLOW would otherwise collapse into a raw ELOOP passthrough error.
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return false, &ahoyError{"refusing to overwrite symlinked .gitignore"}
	}

	// ReadGuarded validates regular-file + size cap on the SAME descriptor it
	// reads, closing the Lstat->ReadFile swap window (iss-109). Its sentinels map
	// back to the existing refusals, and os.IsNotExist keeps the fresh-write path.
	raw, err := fsutil.ReadGuarded(path, gitignoreMaxBytes)
	switch {
	case err == nil:
		// existing regular file within cap — fall through to the round-trip below
	case os.IsNotExist(err):
		eol := "\n"
		body := strings.Join(canonicalGitignoreBlock(entries), eol) + eol
		if werr := fsutil.WriteFileAtomicPreserveMode(path, []byte(body)); werr != nil {
			return false, werr
		}
		return true, nil
	case errors.Is(err, fsutil.ErrNotRegular):
		return false, &ahoyError{"refusing to overwrite non-regular .gitignore"}
	case errors.Is(err, fsutil.ErrTooBig):
		return false, &ahoyError{"refusing to overwrite oversize .gitignore"}
	default:
		return false, err
	}

	eol := gitignoreEOL(raw)
	withoutBlocks := removeGitignoreBlocks(string(raw), eol)
	trimmedLeft := strings.TrimRight(withoutBlocks, "\r\n")
	canonical := strings.Join(canonicalGitignoreBlock(entries), eol) + eol

	var newText string
	if trimmedLeft != "" {
		newText = trimmedLeft + eol + eol + canonical
	} else {
		newText = canonical
	}
	newBytes := []byte(newText)
	if bytes.Equal(newBytes, raw) {
		return false, nil // byte-identical — preserve mtime
	}
	if err := fsutil.WriteFileAtomicPreserveMode(path, newBytes); err != nil {
		return false, err
	}
	return true, nil
}

// removeGitignoreBlocks strips every BEGIN..END block (inclusive) plus any
// stray END without a BEGIN, preserving all other content.
func removeGitignoreBlocks(text, eol string) string {
	if !strings.Contains(text, gitignoreBegin) && !strings.Contains(text, gitignoreEnd) {
		return text
	}
	lines := splitKeepEOL(text)
	var out []string
	var buffered []string // lines held since an as-yet-unclosed BEGIN
	inside := false
	for _, line := range lines {
		stripped := strings.TrimRight(strings.TrimRight(line, "\r\n"), " \t")
		if !inside {
			if stripped == gitignoreBegin {
				inside = true
				buffered = nil
				continue
			}
			if stripped == gitignoreEnd {
				continue // stray END — drop
			}
			out = append(out, line)
			continue
		}
		if stripped == gitignoreEnd {
			inside = false
			buffered = nil // matched BEGIN..END span removed in full
			continue
		}
		buffered = append(buffered, line) // hold until we know the block closes
	}
	// An unbalanced BEGIN with no matching END must NOT swallow everything to EOF:
	// that would silently delete the user's own ignore rules. Mirror the stray-END
	// policy — drop the orphan BEGIN line alone and preserve the content after it.
	if inside {
		out = append(out, buffered...)
	}
	return strings.Join(out, "")
}

// splitKeepEOL splits text into lines that each retain their terminator.
func splitKeepEOL(text string) []string {
	var out []string
	idx := 0
	for idx < len(text) {
		nl := strings.IndexByte(text[idx:], '\n')
		if nl == -1 {
			out = append(out, text[idx:])
			break
		}
		out = append(out, text[idx:idx+nl+1])
		idx += nl + 1
	}
	return out
}

// ahoyError is a small local error type so the package needs no fmt import here.
type ahoyError struct{ msg string }

func (e *ahoyError) Error() string { return e.msg }
