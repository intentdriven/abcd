package launch

// commandladder_test.go — the binary-resolution detector for the command surface
// (iss-205).
//
// # What this guards
//
// `hooks/bootstrap.sh` provisions a checksum-verified binary INTO THE PLUGIN
// ROOT, and itd-105 deferred PATH to `ahoy`. So a by-the-book plugin install
// leaves nothing on PATH, and a command file that reaches for a bare `abcd`
// reaches for something that is not there. The command surface must therefore
// name `"${CLAUDE_PLUGIN_ROOT}/abcd"` FIRST.
//
// # What the record supports, and what it does not
//
// Stated precisely, because a comfortable warrant nobody checked is exactly how
// spc-21 caused iss-204. The plugins reference quoted in iss-205 covers "Skill
// and agent content: anywhere the placeholder appears". `.abcd/work/DECISIONS.md`
// (2026-08-02, iss-170) records only that the harness sets CLAUDE_PLUGIN_ROOT
// inside HOOK invocations, and adds in as many words that this codebase has no
// independent evidence for whether any other context also sets it.
//
// That the same substitution reaches COMMAND content is therefore an
// EXTRAPOLATION from those two, not something this repository has verified. It is
// plausible — a command file is the same class of plugin-bundled markdown — and
// it is the plan's premise, but it stays PENDING the §4 manual install gate,
// which asserts that a bare /abcd resolves via ${CLAUDE_PLUGIN_ROOT} sub-second
// with `go` removed from PATH. Until §4 passes, read this detector as pinning the
// TEXT CONTRACT only: it proves the files say the right thing, never that the
// harness substitutes what they say.
//
// The `go run ./cmd/abcd` rung is worse than slow: under the curated payload
// there is no `cmd/`, no `go.mod` and no source at all, so for a plugin user it
// is an instruction that cannot be followed. It survives only as an explicitly
// source-checkout-qualified note for contributors.
//
// # Why it lives here, as a test
//
// A gate nobody runs is dead scaffolding. `go test ./...` is already the
// preflight and CI authority (`make preflight`, `.github/workflows/ci.yml`), so
// a test in an existing internal package is wired by construction — no Makefile
// target and no workflow step to forget. The launch package is the right one:
// it already owns "would the payload about to be published actually install and
// work" (installsurface.go), and binary resolution in the shipped command files
// is exactly that question asked of the command surface.
//
// # Scope, stated so a later reader does not mistake a gap for an oversight
//
// The detector judges EXECUTABLE instruction — lines inside fenced code blocks,
// which is what an agent copies and runs — plus every `go run ./cmd/abcd`
// mention anywhere in the file. Descriptive inline prose ("`abcd lint` can also
// gate a repo's CI") is deliberately out of scope: it names a capability rather
// than handing over a command line, and policing it would buy nothing while
// making the prose unreadable.
//
// A command file that never names the binary at all (the corpus commands, which
// drive scripts under `~/.abcd.noindex/sources/bin/`) has no ladder to get wrong, so
// every rule below is vacuous for it rather than a failure.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	// pluginRootRung is rung 1: the binary a plugin install actually provisions.
	pluginRootRung = "${CLAUDE_PLUGIN_ROOT}/abcd"
	// goRunRung is rung 3, valid only in a source checkout of this repo.
	goRunRung = "go run ./cmd/abcd"
	// ladderAnchor is the stable marker for the resolution paragraph. Anchoring
	// on it means the ladder cannot be quietly deleted from a file that still
	// invokes the binary.
	ladderAnchor = "**Binary resolution.**"
	// sourceCheckoutQualifier is the qualification rung 3 must carry.
	sourceCheckoutQualifier = "source checkout"
	// qualifierWindow is how far either side of a `go run` mention the
	// qualification may sit — wide enough to survive rewrapping, narrow enough
	// that an unrelated paragraph cannot satisfy it.
	qualifierWindow = 400
)

// refKind classifies one binary reference found in a command file.
type refKind string

const (
	refPluginRoot refKind = "plugin-root"
	// refBarePath is a fenced command line whose invoked program is bare `abcd`
	// — the PATH rung handed over as something to run.
	refBarePath refKind = "bare-path"
	refGoRun    refKind = "go-run"
)

// binaryRef is one reference to the abcd binary, carrying the byte offset that
// orders it against the others.
type binaryRef struct {
	kind   refKind
	offset int
	line   int
	text   string
	fenced bool
}

// shellSeparators splits a fenced line into the segments a shell would treat as
// separate commands, so a piped invocation (`printf … | abcd banlist add …`) is
// judged as the invocation it is rather than hidden behind its first word.
var shellSeparators = regexp.MustCompile(`\|\||&&|[|;(]`)

// commandFiles returns every markdown file under commands/, repo-relative.
func commandFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	dir := filepath.Join(root, "commands")
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".md") {
			rel, rerr := filepath.Rel(root, p)
			if rerr != nil {
				return rerr
			}
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk commands/: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("commands/ holds no markdown files — the detector would pass vacuously")
	}
	return out
}

// invokesBareBinary reports whether a fenced line hands over a command line
// whose invoked program is the bare `abcd` name. Splitting on shell separators
// first catches a piped invocation; comparing the first token exactly keeps a
// path that merely contains the name (`~/.abcd.noindex/sources/bin/add-source`) from
// reading as one.
func invokesBareBinary(line string) bool {
	for _, seg := range shellSeparators.Split(line, -1) {
		fields := strings.Fields(seg)
		if len(fields) > 0 && fields[0] == "abcd" {
			return true
		}
	}
	return false
}

// collectRefs finds every binary reference in one command file, in source order.
//
// Fence tracking is line-based on a leading ``` marker, which is all these files
// use today.
//
// KNOWN BLIND SPOT, measured rather than assumed: an invocation inside a tilde
// fence (~~~) or a four-space-indented code block is invisible to this scan. It
// yields NO ref, so the file passes SILENTLY — and if that file also carries the
// standard Binary-resolution paragraph, all four rules go green over a command
// surface that still hands the agent a bare `abcd`. Both shapes were planted
// under commands/ and watched to pass. A file adopting either needs this taught
// about them first; nothing here will announce the gap.
func collectRefs(body string) []binaryRef {
	var refs []binaryRef

	// Rung 1 and rung 3 are literal strings, findable anywhere in the file.
	for _, lit := range []struct {
		s    string
		kind refKind
	}{{pluginRootRung, refPluginRoot}, {goRunRung, refGoRun}} {
		for i := 0; ; {
			j := strings.Index(body[i:], lit.s)
			if j < 0 {
				break
			}
			off := i + j
			refs = append(refs, binaryRef{
				kind: lit.kind, offset: off,
				line: 1 + strings.Count(body[:off], "\n"), text: lit.s,
			})
			i = off + len(lit.s)
		}
	}

	// Rung 2 handed over as an executable line is a fenced-content concern only.
	offset, inFence := 0, false
	for i, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			offset += len(line) + 1
			continue
		}
		if inFence && invokesBareBinary(line) {
			refs = append(refs, binaryRef{
				kind: refBarePath, offset: offset, line: i + 1,
				text: strings.TrimSpace(line), fenced: true,
			})
		}
		offset += len(line) + 1
	}

	// Mark which go-run mentions sit inside a fence; a fenced one is an
	// instruction rather than a note, and rung 3 is never an instruction.
	markFenced(body, refs)

	// Source order is the whole point of the ordering rule.
	for i := 1; i < len(refs); i++ {
		for j := i; j > 0 && refs[j].offset < refs[j-1].offset; j-- {
			refs[j], refs[j-1] = refs[j-1], refs[j]
		}
	}
	return refs
}

// markFenced sets fenced on every literal reference that lies inside a fence.
func markFenced(body string, refs []binaryRef) {
	offset, inFence := 0, false
	for _, line := range strings.Split(body, "\n") {
		end := offset + len(line)
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			offset = end + 1
			continue
		}
		if inFence {
			for i := range refs {
				if refs[i].kind != refBarePath && refs[i].offset >= offset && refs[i].offset < end {
					refs[i].fenced = true
				}
			}
		}
		offset = end + 1
	}
}

// TestCommandSurfaceResolvesBinaryFromPluginRoot is the iss-205 detector: no file
// under commands/ may name the abcd binary without the plugin-root rung coming
// first, no fenced instruction may hand over a non-plugin-root invocation, and
// the `go run` rung must be qualified as source-checkout-only wherever it
// appears.
func TestCommandSurfaceResolvesBinaryFromPluginRoot(t *testing.T) {
	root := repoRootForTest(t)
	for _, rel := range commandFiles(t, root) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		body := string(data)
		refs := collectRefs(body)
		if len(refs) == 0 {
			continue // names no binary; there is no ladder to get wrong
		}

		// Rule 1 — the plugin-root rung comes first.
		if first := refs[0]; first.kind != refPluginRoot {
			t.Errorf("%s:%d: the first reference to the abcd binary is the %s rung (%q); a plugin install provisions the binary only into the plugin root, so %q must come first",
				rel, first.line, first.kind, first.text, pluginRootRung)
		}

		// Rule 2 — no fenced instruction resolves the binary any other way.
		for _, r := range refs {
			if !r.fenced || r.kind == refPluginRoot {
				continue
			}
			t.Errorf("%s:%d: a fenced command line invokes the binary as %s (%q); fenced lines are what an agent runs verbatim, so they must invoke %q",
				rel, r.line, r.kind, r.text, pluginRootRung)
		}

		// Rule 3 — the go-run rung is qualified as source-checkout-only.
		for _, r := range refs {
			if r.kind != refGoRun {
				continue
			}
			lo := max(0, r.offset-qualifierWindow)
			hi := min(len(body), r.offset+qualifierWindow)
			if !strings.Contains(strings.ToLower(body[lo:hi]), sourceCheckoutQualifier) {
				t.Errorf("%s:%d: %q is not qualified as applying only in a %s; the curated payload carries no cmd/, so an unqualified rung prints an instruction a plugin user cannot follow",
					rel, r.line, goRunRung, sourceCheckoutQualifier)
			}
		}

		// Rule 4 — the ladder itself is present and cannot be silently dropped.
		if !strings.Contains(body, ladderAnchor) {
			t.Errorf("%s: names the abcd binary but carries no %q paragraph; the ladder is what tells a reader what to do when the plugin root is absent",
				rel, ladderAnchor)
		}
	}
}
