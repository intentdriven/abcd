// Package gitleaks is an OPT-IN external-scanner adapter for the transcript
// redaction path (iss-96). It is off by default and adds nothing — no
// dependency invoked, no process spawned, no cost — until a repo asks for it by
// dropping an enabled .abcd/config/gitleaks.json. This realises abcd's
// host-delegated boundary (AGENTS.md: "native/CLI/API/MCP oracles are opt-in
// adapters"): the always-on native scanner (internal/adapter/scanner) stays the
// default, and a repo that wants deeper coverage over the unstructured prose of
// a captured transcript arms this adapter, which shells out to a gitleaks binary
// and folds its findings into the same redaction the native scanner already
// runs.
//
// Reach and cost. The native pattern set is prefix-anchored and misses
// unanchored, labelled, high-entropy values in prose (iss-96's residue).
// gitleaks' generic-api-key rule reaches keyword+delimiter+entropy values that
// the native set does not. Its findings AUGMENT native redaction — they are
// masked by the same scanner.Redact discipline and counted in the record's
// audit counters — they never replace it.
//
// Fail-closed, never a silent no-op. When a repo has opted in but the gitleaks
// binary is not found (not on PATH and no valid configured path), Scan returns
// ErrConfiguredNotFound rather than quietly skipping the deeper scan: a repo
// that armed the adapter must not believe it is covered when it is not. The
// history store surfaces that error and refuses the write, exactly as it fails
// closed on a degraded native scanner.
//
// Binary admission (GHSA-fg9r-3f8g-89m6). The config that names the binary is
// COMMITTED content, so it is trusted for nothing: a candidate binary — a
// configured path or a PATH lookup result alike — runs only when it is an
// absolute path to a regular executable named gitleaks that lies outside the
// repository both lexically and after symlink resolution (admitBinary). A
// refusal is ErrConfiguredPathRefused, as loud as the not-found case and never a
// fallback to PATH, and it names the remedy: a repository rooted at $HOME (a
// dotfiles checkout) contains ~/.local/bin and ~/go/bin, so an install there is
// "inside the repository" and must move outside it, or the path must be unset.
//
// Testability. The binary lookup and the process execution are both injected
// (Adapter.LookPath, Adapter.Runner), so the gate, the loud-stage and the
// augmentation path are all exercised with a fake — no real gitleaks binary is
// required to test this package.
package gitleaks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// configRelPath is the per-repo opt-in config, a sibling of the native
// scanner's .abcd/config/pii.json.
var configRelPath = filepath.Join(".abcd", "config", "gitleaks.json")

// maxConfigBytes caps the per-repo opt-in config (trust boundary), matching the
// native scanner's own override cap.
const maxConfigBytes = 256 * 1024

// binaryName is the only file name a candidate binary may carry (its
// configured spelling, not its symlink target): the name looked up on PATH and
// the name admitBinary requires of a configured path.
const binaryName = "gitleaks"

// runTimeout bounds a single gitleaks invocation so an opted-in repo cannot be
// wedged by a hung binary (the SessionEnd hook holds the history repo lock
// across the capture — the same concern that guards the native config read).
const runTimeout = 30 * time.Second

// ErrConfiguredNotFound is returned when the repo opted in but no gitleaks
// binary could be located. It is deliberately loud: the message names the
// opt-in so an operator sees "gitleaks configured but not found" rather than a
// silent skip.
var ErrConfiguredNotFound = errors.New("gitleaks configured but not found")

// ErrConfiguredPathRefused is returned when a binary WAS located but fails the
// admission rule (admitBinary): it is relative, lies inside the repository, is
// reached through a symlink that does, is not a regular file, or is not
// executable. It is distinct from ErrConfiguredNotFound because the operator's
// remedy differs — the file exists; it is where it is that is the problem — and
// it is equally loud: the history store fails closed on it, and the adapter
// never falls back to PATH after refusing a configured path.
var ErrConfiguredPathRefused = errors.New("gitleaks configured path refused")

// ErrFindingNotLocated is returned when gitleaks reported a finding whose
// Secret and Match both occur nowhere in the scanned text, so there is no span
// to redact. It is loud for the reason ErrConfiguredNotFound is: a repo that
// armed the adapter must not store a transcript the external scanner flagged
// while the record counts zero findings (GHSA-j7v5-q7x6-v3rp). The history
// store fails closed on it; the remedy is the rule that reported the value, or
// enabled:false.
var ErrFindingNotLocated = errors.New("gitleaks finding not located in the text")

// Config is the on-disk opt-in shape (.abcd/config/gitleaks.json). Absent file
// means not opted in (the default).
type Config struct {
	SchemaVersion int    `json:"schema_version"`
	Enabled       bool   `json:"enabled"`
	Path          string `json:"path,omitempty"` // optional explicit binary path
}

// Runner executes gitleaks over text and returns its raw JSON report bytes. It
// is the single seam that touches a process, injected so tests need no binary.
type Runner interface {
	Run(ctx context.Context, binPath, text string) ([]byte, error)
}

// Adapter augments native redaction with gitleaks findings. Construct it with
// NewDefault for the production wiring (real PATH lookup, real exec runner), or
// build one directly with a fake LookPath/Runner in tests.
type Adapter struct {
	// LookPath resolves a bare binary name to a full path (defaults to
	// exec.LookPath). It returns an error when the binary is absent.
	LookPath func(string) (string, error)
	// Runner executes the located binary over the text.
	Runner Runner
}

// NewDefault returns the production adapter: exec.LookPath for discovery and a
// real gitleaks shell-out for execution.
func NewDefault() *Adapter {
	return &Adapter{LookPath: exec.LookPath, Runner: execRunner{}}
}

// LoadConfig reads the per-repo opt-in config with the same trust-boundary
// discipline the native scanner applies to pii.json: an os.Root-contained,
// size-capped, symlink-refusing read. An ABSENT config is the default-off case
// and returns a disabled Config with no error — the whole point of opt-in is
// that a repo which did nothing pays nothing. A PRESENT but unreadable or
// invalid config fails closed with an error, so a repo that tried to arm the
// adapter and got it wrong is told, not silently left unprotected.
func LoadConfig(repoRoot string) (Config, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Config{}, fmt.Errorf("gitleaks: repo root cannot be opened for contained reads: %w", err)
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, configRelPath, maxConfigBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil // not opted in — the default
		}
		return Config{}, fmt.Errorf("gitleaks: per-repo config unreadable (%s): %w", configRelPath, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("gitleaks: per-repo config is not valid JSON (%s): %w", configRelPath, err)
	}
	return cfg, nil
}

// Scan is the transcript-path entry point the history store calls. It loads the
// per-repo config and delegates to the default adapter. When the repo has NOT
// opted in it returns (nil, nil) having invoked nothing.
func Scan(repoRoot, text, logical string) ([]scanner.Finding, error) {
	cfg, err := LoadConfig(repoRoot)
	if err != nil {
		return nil, err
	}
	return NewDefault().Augment(context.Background(), repoRoot, cfg, text, logical)
}

// Augment scans text with gitleaks when cfg opts in, returning findings to fold
// into the native redaction. Behaviour by state:
//
//   - not opted in (cfg.Enabled false): returns (nil, nil), invokes NOTHING —
//     no lookup, no process. This is the gate that keeps the default path free.
//   - opted in, binary absent: returns ErrConfiguredNotFound (loud-stage).
//   - opted in, binary present but inadmissible (admitBinary): returns
//     ErrConfiguredPathRefused (loud-stage), with no fallback to PATH.
//   - opted in, binary present and admitted: runs it and converts its findings;
//     a report it cannot place in the text is ErrFindingNotLocated (loud-stage),
//     never a dropped finding.
func (a *Adapter) Augment(ctx context.Context, repoRoot string, cfg Config, text, logical string) ([]scanner.Finding, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	bin, err := a.resolveBinary(repoRoot, cfg)
	if err != nil {
		return nil, err
	}
	raw, err := a.Runner.Run(ctx, bin, text)
	if err != nil {
		return nil, fmt.Errorf("gitleaks: scan failed: %w", err)
	}
	reports, err := parseReport(raw)
	if err != nil {
		return nil, fmt.Errorf("gitleaks: cannot parse report: %w", err)
	}
	return toFindings(text, logical, reports)
}

// resolveBinary finds the gitleaks binary and admits it under admitBinary. A
// configured path is judged on its own and NEVER falls back to PATH when it is
// refused or missing (a refused config must be fixed, not silently routed
// around); with no configured path the bare name is looked up on PATH and the
// result is held to the same rule. An unresolvable binary on an opted-in repo is
// ErrConfiguredNotFound, an inadmissible one ErrConfiguredPathRefused — never a
// silent skip.
//
// Trust decision on PATH (CWE-426). The configured path comes from a COMMITTED
// file, so it is repository content and is trusted for nothing: admitBinary
// decides. PATH is the operator's process environment, which abcd already
// trusts to locate git, gh and grep everywhere else; a hostile checkout cannot
// set it, and an operator whose PATH is compromised has lost before abcd runs.
// What a checkout CAN reach is a PATH entry that points into it (a relative
// entry, or a per-directory tool shim naming $PWD/bin), so the lookup result is
// admitted exactly as a configured path is — absolute, resolved outside the
// repository, regular, executable — rather than trusted because PATH said so.
func (a *Adapter) resolveBinary(repoRoot string, cfg Config) (string, error) {
	if p := strings.TrimSpace(cfg.Path); p != "" {
		return admitBinary(repoRoot, p, "configured path")
	}
	bin, err := a.LookPath(binaryName)
	if err != nil {
		return "", fmt.Errorf("%w: not on PATH and no path configured", ErrConfiguredNotFound)
	}
	return admitBinary(repoRoot, bin, "PATH lookup")
}

// caseFoldingFS is the package's view of fsutil.CaseFoldingFS, a var so a test
// can provoke the case-folding branch of the containment check on a
// case-sensitive host — the seam ahoy, launch and lifeboat use for the same
// reason.
var caseFoldingFS = fsutil.CaseFoldingFS

// admitBinary is the allow-shape a candidate binary must fit before it is
// executed: an ABSOLUTE path, naming a REGULAR file that is EXECUTABLE, whose
// lexical location AND fully symlink-resolved location both lie OUTSIDE the
// repository root (itself symlink-resolved). Anything else is refused; there is
// no deny-list to slip past. The path returned is the RESOLVED one, so what is
// judged and what is executed are the same bytes.
//
// Both locations are checked because each catches what the other misses: a
// committed symlink under the checkout targeting a real binary elsewhere
// resolves outside but IS repository content (the link is the attacker's), and a
// path outside the checkout whose target resolves into it is repository content
// by another route. Containment is judged twice more, for the same reason: the
// lexical compare routes through fsutil.PathWithin, the canonical one, so on a
// case-folding filesystem a respelt root ("…/REPO/…", or the NFD form of a
// non-ASCII name) that addresses the SAME directory is still "inside"; and an
// identity walk asks the filesystem itself whether any ancestor of the
// candidate IS the repository root (os.SameFile), which holds for every
// spelling a volume folds, whatever a string compare misses.
//
// The last check is the NAME: the location rule says where a binary may be,
// not what it is, and a committed {"path":"/usr/bin/env"} fits every other
// shape — abcd would spawn a program of the attacker's choosing (argv fixed,
// cwd private, output discarded, so program selection only). The CONFIGURED
// spelling must be named gitleaks; the resolved name is not judged, so a
// Homebrew Cellar symlink keeps working.
//
// The executable check is POSIX-only: a non-executable file handed to exec
// fails anyway, but later, with a message that reads like a broken install
// rather than a refused config. On Windows os.Stat synthesises a mode with no
// execute bit, so the check would refuse everything there; windows is not a
// build target, and a port would need to guard it. A repository root that is
// empty or relative cannot anchor the compare and is refused, so the rule
// fails closed rather than trusting a caller that forgot to say where it is.
func admitBinary(repoRoot, candidate, origin string) (string, error) {
	refuse := func(why string) (string, error) {
		return "", fmt.Errorf("%w: %s %q %s", ErrConfiguredPathRefused, origin, candidate, why)
	}
	if repoRoot == "" || !filepath.IsAbs(repoRoot) {
		return refuse("cannot be judged without an absolute repository root")
	}
	if !filepath.IsAbs(candidate) {
		return refuse("is not an absolute path")
	}
	rootLexical := filepath.Clean(repoRoot)
	rootResolved, err := filepath.EvalSymlinks(rootLexical)
	if err != nil {
		return refuse("cannot be judged: repository root does not resolve")
	}
	candLexical := filepath.Clean(candidate)
	candResolved, err := filepath.EvalSymlinks(candLexical)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s %q does not exist", ErrConfiguredNotFound, origin, candidate)
		}
		return refuse("does not resolve")
	}
	const inside = "lies inside the repository; move the binary outside the repository " +
		"(a repository rooted at $HOME contains ~/.local/bin and ~/go/bin), or unset path to use PATH"
	fold := caseFoldingFS()
	for _, root := range []string{rootLexical, rootResolved} {
		for _, cand := range []string{candLexical, candResolved} {
			if fsutil.PathWithin(cand, root, fold) {
				return refuse(inside)
			}
		}
	}
	rootInfo, err := os.Stat(rootResolved)
	if err != nil {
		return refuse("cannot be judged: repository root cannot be stat'ed")
	}
	if hasAncestor(candResolved, rootInfo) {
		return refuse(inside)
	}
	fi, err := os.Stat(candResolved)
	if err != nil {
		return refuse("cannot be stat'ed")
	}
	if !fi.Mode().IsRegular() {
		return refuse("is not a regular file")
	}
	if fi.Mode().Perm()&0o111 == 0 {
		return refuse("is not executable")
	}
	if filepath.Base(candLexical) != binaryName {
		return refuse("is not a gitleaks binary (a file named " + binaryName + " is required)")
	}
	return candResolved, nil
}

// hasAncestor reports whether path, or any directory above it up to the volume
// root, is the same file as root — identity, not spelling. path is already
// symlink-resolved, so the walk crosses no links; a directory that cannot be
// stat'ed is treated as a match, the closed direction for a gate.
func hasAncestor(path string, root os.FileInfo) bool {
	for {
		fi, err := os.Stat(path)
		if err != nil || os.SameFile(fi, root) {
			return true
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false
		}
		path = parent
	}
}

// report is the subset of a gitleaks JSON finding this adapter consumes. Secret
// is the raw credential value gitleaks isolated; Match is the wider surrounding
// text. RuleID names the rule (e.g. generic-api-key).
type report struct {
	RuleID string `json:"RuleID"`
	Secret string `json:"Secret"`
	Match  string `json:"Match"`
}

// parseReport decodes gitleaks' JSON report. An empty report (no findings) is
// valid and yields no reports.
func parseReport(raw []byte) ([]report, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	var reports []report
	if err := json.Unmarshal([]byte(trimmed), &reports); err != nil {
		return nil, err
	}
	return reports, nil
}

// toFindings converts gitleaks reports into scanner.Findings positioned against
// the scanned text, so scanner.Redact masks them exactly as it masks a native
// secret (byte-span fingerprint). It locates each reported value by CONTENT in
// the whole text — the reported Secret, or Match when the Secret does not occur
// verbatim (a rule that reports a decoded credential while the transcript holds
// the encoded form) — rather than trusting gitleaks' line/column indexing,
// which keeps the byte column exact for sealLine and is robust across gitleaks
// versions. Every occurrence gets a finding — a value repeated on several
// lines, or echoed twice on one (a request with its response, a retry log), is
// redacted at each — so after Redact no occurrence of a located value remains,
// which is what lets the store verify an augmented finding by its bytes.
//
// A value that spans lines is split into one finding per line it crosses, each
// carrying that line's fragment, so the line-scoped sealLine masks every line
// of it. gitleaks' default private-key rule reports the key BODY this way, and
// the native pem_private_key pattern matches the BEGIN header only, so this
// split is the one thing that sees the body when the adapter is armed; a
// custom multi-line rule is covered the same way. A report that locates
// nothing is ErrFindingNotLocated: this adapter never drops a finding it was
// handed (GHSA-j7v5-q7x6-v3rp).
//
// Each of those lines is located INDEPENDENTLY across the whole text, not only
// where the whole value sits, because the store verifies an augmented finding
// by its bytes anywhere in the redacted text. Detection and verification then
// share one scope: a line of the value that also occurs benignly elsewhere — a
// quoted end-of-key marker, a repeated header — becomes its own finding and is
// sealed there too, instead of surviving as a span nobody reported and refusing
// the capture forever while the unredacted staged copy waits for a drain that
// can never succeed (iss-2609020231145566). The cost is over-redaction: a
// boilerplate line of a reported value is masked wherever it appears and is
// counted in the record's audit bucket. The value is located only when EVERY
// non-blank line of it is found, so the fail-closed contract is unchanged — a
// text holding just part of the reported value falls through to Match and then
// to ErrFindingNotLocated.
func toFindings(text, logical string, reports []report) ([]scanner.Finding, error) {
	lines := strings.Split(text, "\n")
	// Byte offset at which each line starts, to map a whole-text hit to a
	// (line, column) the line-scoped redactor understands.
	starts := make([]int, len(lines))
	for i, off := 0, 0; i < len(lines); i++ {
		starts[i] = off
		off += len(lines[i]) + 1
	}
	lineAt := func(pos int) int {
		// The last line starting at or before pos.
		return sort.Search(len(starts), func(i int) bool { return starts[i] > pos }) - 1
	}
	// One Finding per (line, column, value) span: two reports of the same
	// value each locate every occurrence, so the same span must not be
	// reported once per report.
	type span struct {
		line, col int
		value     string
	}
	seen := map[span]bool{}
	var out []scanner.Finding
	// locate returns one candidate finding per occurrence of every non-blank
	// LINE of needle, searched independently across the whole text. Per LINE
	// because the redactor is line-scoped; independently because the store
	// verifies a finding by its bytes anywhere in the redacted text, so a line
	// of the value that also occurs benignly elsewhere has to be sealed there
	// too or verification fails on a span detection never reported
	// (iss-2609020231145566). It reports located only when EVERY non-blank line
	// was found: a value the text holds only part of is not this needle, and
	// falls through to the next one rather than half-locating.
	locate := func(needle, kind string) ([]scanner.Finding, bool) {
		var cand []scanner.Finding
		any := false
		for _, frag := range strings.Split(needle, "\n") {
			if strings.TrimSpace(frag) == "" {
				continue // nothing to seal
			}
			any = true
			hits := 0
			for from := 0; from < len(text); {
				idx := strings.Index(text[from:], frag)
				if idx < 0 {
					break
				}
				hit := from + idx
				from = hit + len(frag)
				i := lineAt(hit)
				cand = append(cand, scanner.Finding{
					File:     logical,
					Line:     i + 1,
					Column:   hit - starts[i] + 1, // 1-based byte column, matching sealLine
					Kind:     kind,
					Severity: scanner.SeverityHardFail,
					Matched:  frag,
					Snippet:  lines[i],
				})
				hits++
			}
			if hits == 0 {
				return nil, false
			}
		}
		return cand, any
	}
	for _, r := range reports {
		kind := "gitleaks:" + r.RuleID
		if r.RuleID == "" {
			kind = "gitleaks:generic"
		}
		located := false
		for _, needle := range []string{r.Secret, r.Match} {
			if needle == "" {
				continue
			}
			cand, ok := locate(needle, kind)
			if !ok {
				continue
			}
			for _, f := range cand {
				key := span{f.Line, f.Column, f.Matched}
				if seen[key] {
					continue
				}
				seen[key] = true
				out = append(out, f)
			}
			located = true
			break
		}
		if !located {
			return nil, fmt.Errorf("%w: rule %s reported a value that occurs nowhere in the transcript; fix the rule or set enabled:false",
				ErrFindingNotLocated, strings.TrimPrefix(kind, "gitleaks:"))
		}
	}
	return out, nil
}
