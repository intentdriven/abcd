package repolint

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// maxScanBytes caps how much of a tracked file privacy-hygiene will read. A
// committed file that leaks a path in prose or source is small; anything larger
// is a data blob, skipped so a huge (or endless, via a device) file cannot
// exhaust memory. Exposed to tests via MaxScanBytesForTest.
const maxScanBytes = 4 << 20 // 4 MiB

// MaxScanBytesForTest exposes the scan cap to the package's external tests.
func MaxScanBytesForTest() int { return maxScanBytes }

// privacyHygiene scans committed files for content that must never leave the
// machine: absolute local home paths (/Users/<name>, /home/<name>, C:\Users\),
// network identifiers outside the reserved documentation ranges (addresses,
// LAN hostnames, device names), and the harness-leak class — a live agent-session
// URL and a tool's own attribution footer, which reached public artefacts in
// sibling repositories and, model-authored, this one (iss-178). A line carrying
// the waiver escape
// `abcd-lint:allow` (or the legacy `abcd-audit:allow`) is exempt, so a
// deliberately illustrative value can be kept.
//
// The network half is an allowlist inversion and it does NOT live here: the
// patterns are the scanner's canonical set (internal/adapter/scanner/network.go),
// consulted directly so this rule and Stage-1 redaction can never disagree about
// what a leak is. Real-email and private-repo-name detection need a configured
// allowlist and names file and are deferred to a later phase (recorded in the plan).
type privacyHygiene struct{}

// lintWaiver is the current language-agnostic line-scoped escape. Unlike the
// docs-lint HTML-comment form it works in source files too, where `<!-- -->` is
// not valid. It is the spelling the rule's own Fix hint teaches, so the corpus
// converges on one token.
// privacyConfigRel is the per-repo scanner override this rule reports on when it
// cannot be used. It mirrors scanner's own unexported repoConfigRelPath; the
// scanner does not export it, and duplicating one slash-joined literal is
// cheaper than widening that package's surface for a message string.
const privacyConfigRel = ".abcd/config/pii.json"

const lintWaiver = "abcd-lint:allow"

// auditWaiver is the pre-spc-29 spelling, honoured forever: the token lives in
// committed content (this repo's own sources and any managed repo's), so
// retiring it would silently re-flag every existing deliberately-illustrative
// line. Retiring it would fail closed, which is why permanence is a
// compatibility call rather than a security one.
const auditWaiver = "abcd-audit:allow"

var (
	// Absolute local home paths. A username segment after /Users/ or /home/ is
	// required (so a bare "/Users" mention in prose is not flagged), but a trailing
	// separator is NOT: the username itself is the leak, so "/Users/name" and abcd-audit:allow
	// "/home/name" at end-of-line (e.g. `HOME=/home/name`) must be caught. This abcd-audit:allow
	// mirrors the Windows branch, which never required a trailing separator. Only
	// the Windows arm is case-folded: NTFS is case-insensitive and `c:\users\bob`
	// is a common spelling (Python os.path.normcase lowercases the whole path),
	// while folding the POSIX arm would flag ordinary API-route text ("/users/me").
	absPathRe = regexp.MustCompile(`(?:/Users/|/home/)[A-Za-z0-9._-]+|(?i:[A-Za-z]:\\Users\\[A-Za-z0-9._-]+)`)
)

func (privacyHygiene) Meta() RuleMeta {
	return RuleMeta{
		ID:       "privacy-hygiene",
		Severity: SeverityError,
		Fix: "replace the absolute local path with a repo-relative one, and any network identifier with a reserved documentation value " +
			"(RFC 5737/3849/2606/7042, or a persona-derived device name); or add `abcd-lint:allow` on the line if it is deliberately illustrative",
		PolicyInfo: "an absolute local path or a real network identifier in a committed file leaks a username, a machine, or a network layout; " +
			"committed content uses repo-relative paths and reserved documentation identifiers. A live session URL or a tool's own " +
			"attribution footer leaks the same way and overrides the repository's disclosure convention, so both are refused wherever they are committed",
	}
}

func (privacyHygiene) Where(Context) bool { return true }

func (privacyHygiene) Eval(ctx Context) ([]Finding, error) {
	tracked, err := gitutil.TrackedFiles(ctx.RepoRoot)
	if err != nil {
		return nil, err
	}
	// Contain every read to the repo root. os.Root refuses any path component
	// that escapes the root — including via a symlinked intermediate directory —
	// so a hostile working tree cannot redirect the scan at a file outside the
	// audited repo. If the root itself cannot be opened while git has reported
	// tracked files, the scan cannot run over content that exists — surface the
	// error rather than reporting a clean pass (repolint.go:94: "a check that cannot
	// run must not be silently reported as passing").
	root, err := os.OpenRoot(ctx.RepoRoot)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	// The canonical network-identifier set AS THIS REPO CONFIGURES IT: the
	// scanner's merged patterns, so a severity a repo raised in
	// .abcd/config/pii.json is honoured here exactly as it is in Stage-1
	// redaction.
	//
	// The degraded case is REPORTED, not silently absorbed (iss-203). scanner.New
	// returns a nil error on every degradation path — an unreadable, unparseable
	// or uncompilable pii.json all yield a usable scanner marked unavailable — so
	// an `err == nil` guard here is always true and the fallback branch it reads
	// as guarding is dead code. What actually happens on a broken override is
	// that the merge fails, the scanner keeps the built-in defaults, and the
	// repo's RAISED severities vanish. Reporting that is the whole point: a
	// weakened privacy scan that says "conforms" is the didn't-scan-reported-clean
	// shape this rule's contract forbids, and the operator who raised a severity
	// is the one person who would never learn it stopped applying.
	var out []Finding
	patterns := scanner.NetworkPatterns()
	sc, err := scanner.New(ctx.RepoRoot)
	if err != nil {
		return nil, err
	}
	// The harness-leak class joins the network set, and only that class: a live
	// session URL and a tool's attribution footer are content that must not be
	// committed, on the same footing as a leaked address (iss-178). The rest of
	// DefaultPatterns — the API keys and PATs — deliberately stays out, because
	// widening this rule to every secret shape is a different behavioural change
	// with its own false-positive budget over deliberately redacted fixtures.
	//
	// These two come from the package-level set rather than the merged one: the
	// severity is not something a repo has a reason to move (there is no reading
	// under which either belongs in public text), and reading them from the
	// canonical set means the degraded-config path below cannot silently drop
	// them.
	leakPatterns := scanner.HarnessLeakPatterns()
	if degraded, reason := sc.Unavailable(); degraded {
		out = append(out, Finding{
			RuleID:   "privacy-hygiene",
			Severity: SeverityError,
			File:     privacyConfigRel,
			Message: "per-repo scanner config is unusable, so this scan runs with the " +
				"built-in severities and any raised in it do not apply: " + reason,
			Fix: "repair or remove " + privacyConfigRel,
		})
	} else {
		patterns = sc.NetworkPatterns()
	}
	patterns = append(patterns, leakPatterns...)

	for _, rel := range tracked {
		data, ok, oversizeText, openErr := readTrackedFile(root, filepath.FromSlash(rel))
		if !ok {
			// A textual file over the scan cap was NOT scanned, and silence
			// here would report "conforms" for content nobody looked at —
			// the didn't-scan-reported-clean shape the engine contract
			// forbids (iss-356 item 4). Binary blobs stay quiet: the scan
			// would skip them anyway, so the cap loses nothing there.
			// The open branch gets the same treatment (the size branch's fix
			// stopped one arm short): a tracked file the scan cannot OPEN is
			// content nobody looked at, so it warns — except the absent and
			// tracked-symlink shapes, which are legitimate worktree states
			// the scan skips by design.
			if oversizeText {
				out = append(out, Finding{
					RuleID:   "privacy-hygiene",
					Severity: SeverityWarn,
					File:     rel,
					Message: fmt.Sprintf("not scanned: over the %d MiB privacy-scan cap; split the file or verify it by hand",
						maxScanBytes>>20),
				})
			} else if openFailureWarrantsWarn(openErr) {
				out = append(out, Finding{
					RuleID:   "privacy-hygiene",
					Severity: SeverityWarn,
					File:     rel,
					Message:  "not scanned: the tracked file could not be opened; fix its permissions or verify it by hand",
				})
			}
			continue
		}
		if isBinary(data) {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if strings.Contains(line, lintWaiver) || strings.Contains(line, auditWaiver) {
				continue
			}
			msg, fix, sev, leaked := privacyLeak(line, patterns)
			if !leaked {
				continue
			}
			out = append(out, Finding{
				RuleID:   "privacy-hygiene",
				Severity: sev,
				File:     rel,
				Line:     i + 1,
				Message:  msg,
				Fix:      fix,
			})
		}
	}
	return out, nil
}

// privacyLeak reports whether one line carries content that must never be
// committed, the message naming the class, the fix that class calls for, and the
// severity it carries. At most one finding per line is reported (the
// absolute-path class first), matching the rule's v1 behaviour: the citation
// points a reader at the line, and the line is what gets fixed.
//
// The severity comes from the PATTERN, not from the rule: the canonical set
// draws a deliberate line between addresses, which are range-checked and block,
// and the two hostname shapes, which are heuristics and warn. Flattening
// everything to error here would have made that documented split a fiction at
// the one surface a reader meets it.
//
// The fix likewise comes from the pattern where it has one. The rule's own Fix
// speaks about paths and addresses, which is no use to somebody looking at a
// tool footer, and the harness class has an operational half — the post-create
// re-read — that only its own policy states (scanner.OutboundPolicy).
func privacyLeak(line string, patterns []scanner.Pattern) (msg, fix string, sev Severity, leaked bool) {
	if hasAbsHomePath(line) {
		return "committed file contains an absolute local path", "", SeverityError, true
	}
	for _, p := range patterns {
		for _, loc := range p.Re.FindAllStringIndex(line, -1) {
			matched := line[loc[0]:loc[1]]
			if p.Skip != nil && p.Skip(matched) {
				continue
			}
			if p.SkipAt != nil && p.SkipAt(line, loc[0], loc[1]) {
				continue
			}
			if scanner.IsHarnessLeakKind(p.Kind) {
				return "committed file contains a " + p.Label, scanner.OutboundPolicy, lintSeverity(p.Severity), true
			}
			return "committed file contains a " + p.Label, "", lintSeverity(p.Severity), true
		}
	}
	return "", "", SeverityError, false
}

// lintSeverity maps a scanner severity onto the lint surface's two levels. A
// scanner hard_fail blocks; anything softer is advisory. An unknown value maps
// to the blocking level, so a new pattern can never be quieter than intended by
// accident.
func lintSeverity(s scanner.Severity) Severity {
	if s == scanner.SeverityWarn || s == scanner.SeverityInfo {
		return SeverityWarn
	}
	return SeverityError
}

// hasAbsHomePath reports whether a line carries an absolute home path whose
// final segment is a real username. A segment naming a well-known system
// directory under /Users — Shared, Guest, Public — is NOT a username (iss-153),
// so product code that legitimately writes to /Users/Shared needs no waiver. The
// exemption is scoped to the /Users root: a /home/<name> segment is always a
// user, and the allowlist is a macOS convention.
//
// Two further shapes are exempt because the conventions MANDATE them, and a
// detector that is red at baseline on its own conventions is one nobody reads
// (iss-2609100505145554):
//
//   - a username segment that is a persona from the registry, which is what
//     examples and fixtures are required to use;
//   - anything beneath a system root, because a system root is not a home root,
//     so no segment under it sits in the username position at all.
//
// Both narrowings are stated in full at their own call sites below, together
// with what they deliberately stop catching.
func hasAbsHomePath(line string) bool {
	for _, loc := range absPathRe.FindAllStringIndex(line, -1) {
		if !leadingBoundaryOK(line, loc[0]) {
			// The match continues a longer path segment rather than beginning
			// one — an ordinary relative directory ("src/pages/home/x") or a URL
			// path ("https://host/home/x"), not a leaked absolute local path. The
			// scanner twin (genericHomeRe) gates on the same boundary; without it
			// this rule hard-fails at Error on ubiquitous committed content.
			continue
		}
		m := line[loc[0]:loc[1]]
		seg := m
		if i := strings.LastIndexAny(m, `/\`); i >= 0 {
			seg = m[i+1:]
		}
		if isUsersRoot(m) && scanner.IsNonUserHomeSegment(seg) {
			// A system root (/Users/Shared, /Users/Guest, C:\Users\Public) is not
			// a home root: the username position is the segment immediately after
			// /Users or /home, and that position is held here by a directory that
			// names no user. Nothing DEEPER is in the username position either, so
			// the whole subtree is exempt — which is what iss-153 asked for
			// ("/Users/Shared/...") and what the product needs, because it creates
			// such a directory and has to name it in comments, tests and docs.
			//
			// The exemption stops at a TRAVERSAL segment. "/Users/Shared/../bob"
			// and "/Users/Shared//bob" leave the shared root again, so the name
			// after them is back in the username position — this is the half of
			// the old narrowing that was actually load-bearing, and it stays.
			//
			// Deliberately no longer caught: a personal name used as an ordinary
			// directory name inside a shared folder (/Users/Shared/<name>/x). That
			// is not a home path, and this rule detects home paths; the committing
			// user's OWN name there is still caught by the scanner's
			// local_username detector at hard_fail.
			if reachedNameViaTraversal(line, loc[1], isWindowsPath(m)) {
				return true
			}
			continue
		}
		if isPersonaHomeSegment(seg) {
			// The conventions require examples and fixtures to use persona homes,
			// and the rule's own Fix hint already blesses a persona-derived device
			// name; treating the same roster as a leak in a path made the gate
			// permanently red on a conforming repo.
			continue
		}
		return true
	}
	return false
}

// isPersonaHomeSegment reports whether a username segment is a registry persona
// that is NOT this machine's own home user.
//
// The second half is what keeps the exemption honest. A real account for someone
// called Alice is spelled `/Users/alice`, exactly like the fixture, so the roster
// alone cannot separate them. The one machine where the distinction is decidable
// — and the one where it matters most, because your own home path is the leak you
// actually commit — is the machine whose home user IS that name, and there the
// exemption yields. scanner.CallerHome is the same notion of "the caller's own
// home" the scanner's home_path_self detector uses, so there is one answer to it
// rather than two.
//
// Residual, stated plainly: a home path belonging to a DIFFERENT person whose
// account name happens to be one of the roster's given names is no longer flagged
// by this rule. That is the cost the record accepts in exchange for a gate that
// is green on a conforming repo; the alternative it names — a waiver marker on
// every example the conventions asked for — was judged worse.
func isPersonaHomeSegment(seg string) bool {
	if !scanner.IsPersonaName(seg) {
		return false
	}
	home := scanner.CallerHome()
	i := strings.LastIndexAny(home, `/\`)
	if i < 0 {
		return true
	}
	return !strings.EqualFold(home[i+1:], seg)
}

// reachedNameViaTraversal reports whether a NAME-BEARING path segment follows the
// system-directory match at pos WITH A TRAVERSAL SEGMENT IN BETWEEN — a dots-only
// or an empty segment ("/Users/Shared/../<user>", "/Users/Shared//<user>").
//
// That is the one shape where a name beneath a system root is back in the
// username position: the traversal walks out of the shared root, so the segment
// after it is a home directory again, and letting the system directory swallow it
// would hand back the shield the exemption must not give.
//
// Everything else under a system root is exempt. A name reached DIRECTLY
// ("/Users/Shared/<seg>/x") is an ordinary entry inside a shared folder, not a
// home path, and flagging it taxed the product code that has to name its own
// shared directory (iss-2609100505145554). "/Users/Shared" and "/Users/Shared/"
// have no following segment at all, and "/Users/Shared/..." is prose with an
// ellipsis.
//
// The separators the walk accepts come from the MATCH, not from the host it runs
// on. A Windows path takes BOTH: Windows itself accepts either separator and
// mixes them freely within one path, so `C:\Users\Shared/..\<user>` traverses
// exactly as the all-backslash spelling does, and walking on one separator would
// exempt the mixed spelling wholesale. A POSIX path stays slash-only, because a
// backslash after one is an escape (the two bytes of "/Users/Shared\n" in a
// source string), never a path segment.
func reachedNameViaTraversal(line string, pos int, windows bool) bool {
	isSep := func(b byte) bool { return b == '/' || (windows && b == '\\') }
	traversed := false
	for pos < len(line) && isSep(line[pos]) {
		i, named := pos+1, false
		for i < len(line) && isPathSegmentChar(line[i]) {
			if line[i] != '.' {
				named = true
			}
			i++
		}
		if named {
			// A name is a leak only if a traversal segment preceded it. An empty
			// segment counts: "/Users/Shared//<user>" is the doubled-separator
			// spelling of the same escape.
			return traversed
		}
		// The segment names nothing: either pure dots ("." / "..") or empty (two
		// separators in a row). Both are the escape out of the shared root.
		traversed = true
		pos = i // skip the traversal segment and keep looking
	}
	return false
}

// isWindowsPath reports whether the matched path is the Windows spelling
// (`C:\Users\<name>`) rather than a POSIX one.
func isWindowsPath(m string) bool {
	return strings.Contains(strings.ToLower(m), `:\users\`)
}

// leadingBoundaryOK reports whether the match at start BEGINS a path rather than
// continuing a longer segment. A path-segment char immediately before the match
// (the 's' of "pages/home/x", the 'm' of "example.com/home/x") means the
// "/home/" or "/Users/" is interior to a relative path or a URL, not an absolute
// local leak. '/' is NOT a segment char here, so "file:///home/alice" stays
// flagged — deliberately narrower than the scanner's predicate.
func leadingBoundaryOK(line string, start int) bool {
	if start == 0 {
		return true
	}
	return !isPathSegmentChar(line[start-1])
}

// isPathSegmentChar matches the character class absPathRe uses for a segment.
func isPathSegmentChar(b byte) bool {
	return b == '.' || b == '_' || b == '-' ||
		(b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// isUsersRoot reports whether an absolute-path match hangs off a /Users (or
// C:\Users) root rather than /home.
func isUsersRoot(m string) bool {
	l := strings.ToLower(m)
	return strings.HasPrefix(l, "/users/") || strings.Contains(l, `:\users\`)
}

// readTrackedFile reads a tracked path safely for scanning, relative to root
// (an os.Root scoped to the repo, so no component can escape the repo — the
// containment guarantee the leaf-only O_NOFOLLOW could not give). O_NONBLOCK
// makes opening a FIFO or device return immediately rather than block until a
// writer appears; the regular-file check then skips it before any read. It reads
// only regular files and never more than one byte past maxScanBytes, so a huge
// or device-backed file cannot exhaust memory. A file that cannot be opened, is not
// a regular file, or exceeds the cap is skipped (ok=false), not a scan failure;
// oversizeText additionally reports that a skipped file is over the cap yet
// looks textual, so the caller can say "not scanned" instead of staying silent.
// openFailureWarrantsWarn parts the open failures that are legitimate worktree
// states from the ones that mean unscanned content. TrackedFiles lists INDEX
// entries, so an absent path (deleted in the worktree but not yet committed, a
// sparse checkout) is a normal state the scan has nothing to read for, and a
// symlink-shaped refusal (a tracked link leaf under O_NOFOLLOW, an os.Root
// containment refusal) is the scan's own skip-by-design, pinned by the
// symlink tests. A permission or I/O fault is different in kind: the content
// exists, nobody looked at it, and silence would read as "conforms" — the
// same not-scanned shape the oversize arm already warns on (iss-356 item 4).
func openFailureWarrantsWarn(err error) bool {
	return errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EIO)
}

func readTrackedFile(root *os.Root, rel string) (data []byte, ok, oversizeText bool, openErr error) {
	f, err := root.OpenFile(rel, os.O_RDONLY|syscallNoFollow, 0)
	if err != nil {
		// The caller decides whether the failure is warn-worthy
		// (openFailureWarrantsWarn); reporting it is not this helper's call.
		return nil, false, false, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, false, false, nil // FIFO, device, directory, or vanished
	}
	if info.Size() > maxScanBytes {
		// Not scanned — but say whether it LOOKS like prose, so the caller
		// can warn on a skipped textual file without flagging every committed
		// binary asset. 8 KiB is isBinary's own probe horizon.
		probe := make([]byte, 8<<10)
		n, err := io.ReadFull(f, probe)
		if n == 0 {
			// The probe itself failed, so nothing about the file is known —
			// warn rather than stay silent (the not-scanned shape again).
			_ = err
			return nil, false, true, nil
		}
		return nil, false, !isBinary(probe[:n]), nil
	}
	data, ok, oversizeText = capRead(f)
	return data, ok, oversizeText, nil
}

// capRead reads everything the scan may see from an already-vetted regular
// file: one byte past the cap, so a file that grew past maxScanBytes between
// the fstat and the read is detected rather than silently scanned as a
// truncated prefix and reported clean — a false "scanned" on a privacy control
// (the same size TOCTOU the caller's fstat branch handles for a file already
// over the cap). A grown file is refused whole and reported not-scanned, using
// the bytes already in hand for the textual probe rather than re-reading.
func capRead(r io.Reader) (data []byte, ok, oversizeText bool) {
	data, err := io.ReadAll(io.LimitReader(r, maxScanBytes+1))
	if err != nil {
		return nil, false, false
	}
	if int64(len(data)) > maxScanBytes {
		return nil, false, !isBinary(data)
	}
	return data, true, false
}

// isBinary reports whether data looks non-textual: a NUL byte in the first 8 KiB
// is the standard heuristic git itself uses. A binary file cannot leak a path in
// a way this line scanner would read correctly, so it is skipped.
func isBinary(data []byte) bool {
	n := len(data)
	if n > 8192 {
		n = 8192
	}
	return bytes.IndexByte(data[:n], 0) >= 0
}
