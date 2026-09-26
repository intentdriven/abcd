package ahoy

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// markerInner is the canonical inner content of the marker block (the rule
// loader text). Drift detection is only meaningful because there is one
// canonical source; it is embedded so the binary is self-contained.
//
//go:embed defaults/claude-md-marker-block.md
var markerInner []byte

// Canonical marker fences. Kept as byte constants for byte-level idempotency.
var (
	markerBegin = []byte("<!-- BEGIN ABCD -->")
	markerEnd   = []byte("<!-- END ABCD -->")
)

// markerBlockRe matches one fenced block, non-greedy across newlines. It only
// matches a balanced BEGIN...END pair, so an unbalanced fence reads as absent.
var markerBlockRe = regexp.MustCompile(`(?s)` + regexp.QuoteMeta("<!-- BEGIN ABCD -->") + `(.*?)` + regexp.QuoteMeta("<!-- END ABCD -->"))

// frontmatterRe matches a leading YAML frontmatter block (CRLF-aware, empty ok).
var frontmatterRe = regexp.MustCompile(`(?s)\A---(?:\r?\n)(?:.*?(?:\r?\n))?---(?:\r?\n)`)

// h1Re matches the first ATX-1 heading line (CRLF-aware).
var h1Re = regexp.MustCompile(`(?m)^# .*(?:\r?\n)`)

// detectEOL recovers a file's newline flavour. Absent/no-newline defaults to LF.
func detectEOL(existing []byte) []byte {
	if bytes.Contains(existing, []byte("\r\n")) {
		return []byte("\r\n")
	}
	return []byte("\n")
}

// synthesizeMarker builds the canonical wrapped block for a target EOL. The
// inner template's newlines are normalised to the target EOL first so a CRLF
// file never ends up with mixed EOLs.
func synthesizeMarker(inner, eol []byte) []byte {
	innerNorm := bytes.ReplaceAll(inner, []byte("\r\n"), []byte("\n"))
	innerNorm = bytes.ReplaceAll(innerNorm, []byte("\n"), eol)
	out := make([]byte, 0, len(markerBegin)+len(innerNorm)+len(markerEnd)+2*len(eol))
	out = append(out, markerBegin...)
	out = append(out, eol...)
	out = append(out, innerNorm...)
	out = append(out, eol...)
	out = append(out, markerEnd...)
	return out
}

// markerState is the classifier result.
type markerState string

const (
	markerCurrent  markerState = "current"
	markerOutdated markerState = "outdated"
	markerMissing  markerState = "missing"
	// markerSymlink is a symlinked leaf: installMarkerFile refuses to write
	// through one, so it is neither "missing" (resolvable) nor "current". It is
	// a distinct non-resolvable state so detection never advertises a resolvable
	// gap that apply can never close.
	markerSymlink markerState = "symlink"
	// markerUnplaceable is a file whose block would be appended at end of file
	// inside a fenced block or an HTML comment nothing closes, where no reader
	// sees it as markdown. installMarkerFile refuses to write it, so it is not
	// a resolvable gap either.
	markerUnplaceable markerState = "unplaceable"
)

// classifyMarker reads targetPath and classifies its marker block. A read error
// or absent file is observably equivalent to "missing".
func classifyMarker(targetPath string) markerState {
	// Do not follow a symlinked leaf: installMarkerFile refuses to write through
	// one, so classifying it as "missing" would make detection emit a resolvable
	// gap that install can never close (a permanent "partial").
	if fi, err := os.Lstat(targetPath); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return markerSymlink
	}
	existing, err := fsutil.ReadGuarded(targetPath, maxAhoyFileBytes)
	if err != nil {
		return markerMissing
	}
	matches := markerBlockRe.FindAllIndex(existing, -1)
	if len(matches) == 0 {
		if appendsInsideOpenSpan(existing) {
			return markerUnplaceable
		}
		return markerMissing
	}
	if len(matches) > 1 {
		return markerOutdated
	}
	eol := detectEOL(existing)
	expected := synthesizeMarker(markerInner, eol)
	first := matches[0]
	if bytes.Equal(existing[first[0]:first[1]], expected) {
		return markerCurrent
	}
	return markerOutdated
}

// installMarkerFile plants, updates, or leaves-current the block in one target.
// It returns (wrote, err): a non-nil err is a per-target failure that leaves the
// file untouched, and says why, so the install can tell the person which file
// kept no block and for what reason (iss-2609260057127611). Byte-stable: a
// current block is not rewritten.
func installMarkerFile(targetPath string) (wrote bool, err error) {
	// Reject a symlinked leaf so a planted symlink cannot redirect the write.
	if fi, lerr := os.Lstat(targetPath); lerr == nil && fi.Mode()&os.ModeSymlink != 0 {
		return false, &ahoyError{"it is a symlink, and abcd never writes through one"}
	}
	existing, rerr := fsutil.ReadGuarded(targetPath, maxAhoyFileBytes)
	absent := false
	if rerr != nil {
		if !os.IsNotExist(rerr) {
			return false, fmt.Errorf("it could not be read: %w", rerr)
		}
		absent = true
	}
	eol := detectEOL(existing)
	synth := synthesizeMarker(markerInner, eol)

	if absent {
		body := append(append([]byte{}, synth...), eol...)
		if werr := fsutil.WriteFileAtomicPreserveMode(targetPath, body); werr != nil {
			return false, fmt.Errorf("it could not be written: %w", werr)
		}
		return true, nil
	}

	matches := markerBlockRe.FindAllIndex(existing, -1)
	if len(matches) == 0 {
		if appendsInsideOpenSpan(existing) {
			return false, &ahoyError{"a fenced block or HTML comment in it is never closed, so the block would land inside it"}
		}
		body := composeMarkerInsertion(existing, synth, eol)
		if werr := fsutil.WriteFileAtomicPreserveMode(targetPath, body); werr != nil {
			return false, fmt.Errorf("it could not be written: %w", werr)
		}
		return true, nil
	}
	first := matches[0]
	if len(matches) == 1 && bytes.Equal(existing[first[0]:first[1]], synth) {
		return false, nil // current — no write, mtime preserved
	}
	body := composeMarkerReplacement(existing, matches, synth)
	if werr := fsutil.WriteFileAtomicPreserveMode(targetPath, body); werr != nil {
		return false, fmt.Errorf("it could not be written: %w", werr)
	}
	return true, nil
}

// composeMarkerInsertion inserts synth into a file with no block: after
// frontmatter, else after the first H1, else appended at EOF.
func composeMarkerInsertion(existing, synth, eol []byte) []byte {
	if loc := frontmatterRe.FindIndex(existing); loc != nil {
		head, tail := existing[:loc[1]], existing[loc[1]:]
		return concat(head, eol, synth, trailingSep(tail, eol), tail)
	}
	if end := firstOutOfFenceH1(existing); end != -1 {
		head, tail := existing[:end], existing[end:]
		return concat(head, eol, synth, trailingSep(tail, eol), tail)
	}
	if len(existing) > 0 && !bytes.HasSuffix(existing, eol) && !bytes.HasSuffix(existing, []byte("\n")) {
		existing = append(existing, eol...)
	}
	if len(existing) == 0 {
		return concat(synth, eol)
	}
	return concat(existing, eol, synth, eol)
}

// appendsInsideOpenSpan reports whether composeMarkerInsertion would append the
// block at end of file inside a fenced block or an HTML comment nothing closes:
// there is no frontmatter and no live H1 to place it after, and mdrecord reads
// a span still open at the end. Every line below such an opener is code or a
// comment to every reader, so a block appended there is never read as markdown,
// while the byte regex would still classify it current (iss-2609251522453961).
func appendsInsideOpenSpan(existing []byte) bool {
	if frontmatterRe.Match(existing) || firstOutOfFenceH1(existing) != -1 {
		return false
	}
	lines, _, _ := markerLines(existing)
	_, _, open := mdrecord.Unclosed(lines)
	return open
}

// markerLines splits a file into its lines, line endings trimmed, with each
// line's start and end byte offsets.
func markerLines(existing []byte) (lines []string, starts, ends []int) {
	for offset := 0; offset < len(existing); {
		next := len(existing)
		if nl := bytes.IndexByte(existing[offset:], '\n'); nl != -1 {
			next = offset + nl + 1
		}
		lines = append(lines, strings.TrimRight(string(existing[offset:next]), "\r\n"))
		starts, ends = append(starts, offset), append(ends, next)
		offset = next
	}
	return lines, starts, ends
}

// firstOutOfFenceH1 returns the byte offset just past the first ATX-1 heading
// line that is live markdown, or -1 when none exists. Which lines are live is
// mdrecord's answer, the tree's one CommonMark reading: a `# ` line inside a
// fenced block (e.g. a shell comment) or an HTML comment is not the title, and
// a fence closes only on a bare run of its own marker at least as long as the
// opener, so a quoted fence inside a longer one does not end it early
// (iss-2609250955219864). Placing the block after such a line would write it
// inside the example.
func firstOutOfFenceH1(existing []byte) int {
	lines, starts, ends := markerLines(existing)
	mask := mdrecord.Mask(lines)
	for i := range lines {
		if mask[i] == 0 && h1Re.Match(existing[starts[i]:ends[i]]) {
			return ends[i]
		}
	}
	return -1
}

// trailingSep is the separator between the block and following text: at least
// one EOL, two when the tail is non-empty and does not start with an EOL.
func trailingSep(tail, eol []byte) []byte {
	if len(tail) == 0 || tail[0] == '\n' || tail[0] == '\r' {
		return eol
	}
	return concat(eol, eol)
}

// composeMarkerReplacement replaces the first block with synth and drops later
// blocks (collapse-to-one), preserving text between blocks.
func composeMarkerReplacement(existing []byte, matches [][]int, synth []byte) []byte {
	first := matches[0]
	out := make([]byte, 0, len(existing))
	out = append(out, existing[:first[0]]...)
	out = append(out, synth...)
	cursor := first[1]
	for _, m := range matches[1:] {
		out = append(out, existing[cursor:m[0]]...)
		cursor = m[1]
	}
	out = append(out, existing[cursor:]...)
	return out
}

// removeMarkerFile strips every abcd block from one target, collapsing the EOLs
// install introduced so install->uninstall round-trips. Returns (wrote, err): a
// symlinked leaf, a non-regular file, or a failed read or write leaves the file
// untouched and returns why.
func removeMarkerFile(targetPath string) (wrote bool, err error) {
	fi, lerr := os.Lstat(targetPath)
	if lerr != nil {
		if os.IsNotExist(lerr) {
			return false, nil // absent — nothing to remove
		}
		return false, fmt.Errorf("it could not be examined: %w", lerr)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return false, &ahoyError{"it is a symlink, and abcd never writes through one"}
	}
	if !fi.Mode().IsRegular() {
		return false, &ahoyError{"it is not a regular file"}
	}
	existing, rerr := fsutil.ReadGuarded(targetPath, maxAhoyFileBytes)
	if rerr != nil {
		return false, fmt.Errorf("it could not be read: %w", rerr)
	}
	matches := markerBlockRe.FindAllIndex(existing, -1)
	if len(matches) == 0 {
		return false, nil // no block — untouched
	}
	body := composeMarkerRemoval(existing, matches)
	if werr := fsutil.WriteFileAtomicPreserveMode(targetPath, body); werr != nil {
		return false, fmt.Errorf("it could not be written: %w", werr)
	}
	return true, nil
}

// StripMarkerBlock returns content with every abcd marker block (a balanced
// BEGIN…END fence and its surrounding blank lines) removed, and reports whether
// anything was stripped. Content with no block is returned unchanged. It is the
// pure, path-free form of removeMarkerFile, exported for the lifeboat packer:
// carrying a stale marker block into a packed record would plant a dead
// rules-loader in whatever repo later embarks it, so a verbatim copy is passed
// through this first. Only a balanced pair is matched, so an unbalanced fence or
// a literal mention of the delimiter text outside a pair is left intact.
func StripMarkerBlock(content []byte) ([]byte, bool) {
	matches := markerBlockRe.FindAllIndex(content, -1)
	if len(matches) == 0 {
		return content, false
	}
	return composeMarkerRemoval(content, matches), true
}

// composeMarkerRemoval deletes each block plus its surrounding EOL run and
// reinserts one blank-line separator when both sides survive.
func composeMarkerRemoval(existing []byte, matches [][]int) []byte {
	eol := detectEOL(existing)
	var out []byte
	cursor := 0
	n := len(existing)
	for _, m := range matches {
		start, end := m[0], m[1]
		adjStart := start
		for adjStart > 0 {
			if adjStart >= 2 && bytes.Equal(existing[adjStart-2:adjStart], []byte("\r\n")) {
				adjStart -= 2
				continue
			}
			if c := existing[adjStart-1]; c == '\n' || c == '\r' {
				adjStart--
				continue
			}
			break
		}
		adjEnd := end
		for adjEnd < n {
			if adjEnd+2 <= n && bytes.Equal(existing[adjEnd:adjEnd+2], []byte("\r\n")) {
				adjEnd += 2
				continue
			}
			if c := existing[adjEnd]; c == '\n' || c == '\r' {
				adjEnd++
				continue
			}
			break
		}
		head := existing[cursor:adjStart]
		tail := existing[adjEnd:]
		out = append(out, head...)
		switch {
		case len(head) > 0 && len(tail) > 0:
			out = append(out, eol...)
			out = append(out, eol...)
		case len(head) > 0 && len(tail) == 0:
			if !bytes.HasSuffix(head, eol) && !bytes.HasSuffix(head, []byte("\n")) {
				out = append(out, eol...)
			}
		}
		cursor = adjEnd
	}
	out = append(out, existing[cursor:]...)
	return out
}

// markerTargets maps a docs.target value to the files that host the block.
func markerTargets(docsTarget string) []string {
	switch docsTarget {
	case "claude_md":
		return []string{"CLAUDE.md"}
	case "agents_md":
		return []string{"AGENTS.md"}
	case "skip":
		return nil
	case "both":
		return []string{"CLAUDE.md", "AGENTS.md"}
	default:
		// Unset or unrecognised: the default target. A value nobody chose never
		// names a file, so detection previews exactly what an install with that
		// config would plant — which, at the skip default, is nothing.
		return markerTargets(docsTargetDefault)
	}
}

// markerFileHasBlock reports whether a target file contains at least one BEGIN
// fence (used as a strong managed-repo signal during classification).
func markerFileHasBlock(targetPath string) bool {
	fi, err := os.Lstat(targetPath)
	if err != nil || fi.Mode()&os.ModeSymlink != 0 {
		return false
	}
	data, err := fsutil.ReadGuarded(targetPath, maxAhoyFileBytes)
	if err != nil {
		return false
	}
	return bytes.Contains(data, markerBegin)
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
