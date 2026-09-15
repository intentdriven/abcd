package scanner

import (
	"os"
	"strings"
)

// residual.go — the stage-two discipline the committed stores share.
//
// Every store that writes free text into a committed artefact (the history
// transcript store, the memory page store) redacts through this package and
// then asks the same three questions before the write lands: what is the
// caller's home, did it survive, and which rescan findings must refuse the
// write. Each store used to carry its own copy of the answers, so the stores
// that are meant to agree on what counts as a leak could be fixed apart. The
// answers live here once; the stores call them.

// CallerHome resolves the caller's home directory as ProbeIdentity does — $HOME
// first (so tests and redirected runs agree), then os.UserHomeDir — trimmed of
// any trailing slash. Empty when neither resolves.
func CallerHome() string {
	home := os.Getenv("HOME")
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}
	return strings.TrimRight(home, "/")
}

// BlockingResidual filters a stage-two rescan to the findings that must refuse
// a write. Severity alone is the wrong gate: the hostname patterns are shape
// heuristics and therefore warn by design, so a LAN host or device name that
// survived stage-one redaction would otherwise be committed in silence — the
// very class of leak the stores exist to stop. Any surviving IDENTITY or
// NETWORK span refuses the write whatever its severity; everything else still
// gates on hard_fail. After the stage-one detector fixes this path is rarely
// reachable, which is what a backstop is for.
func BlockingResidual(findings []Finding) []Finding {
	var out []Finding
	for _, f := range findings {
		if f.Severity == SeverityHardFail || IsIdentityKind(f.Kind) {
			out = append(out, f)
		}
	}
	return out
}

// SweepCallerHome is the deterministic literal $HOME backstop, independent of
// the pattern heuristic: every occurrence of home that stands as a path is
// collapsed to "~". An occurrence stands as a path when nothing path-like
// precedes it (the start of the text, or a byte that cannot continue a path
// segment) and nothing name-like follows it (the end, a separator, or any
// byte outside the username-continuation set). An unanchored replace turned
// "/rootfs/etc/hosts" into "~fs/etc/hosts" under HOME=/root and "/home/abc/x"
// into "~bc/x" under HOME=/home/a, silently corrupting the committed text; the
// anchor is what lets a short home coexist with the paths that merely share
// its prefix. An empty home sweeps nothing.
func SweepCallerHome(text, home string) string {
	if home == "" {
		return text
	}
	urls := urlSpans(text)
	var b strings.Builder
	from := 0
	for {
		i := strings.Index(text[from:], home)
		if i < 0 {
			break
		}
		at := from + i
		end := at + len(home)
		if homeSweepable(text, at, end, urls) {
			b.WriteString(text[from:at])
			b.WriteByte('~')
			from = end
			continue
		}
		b.WriteString(text[from : at+1])
		from = at + 1
	}
	if from == 0 {
		return text
	}
	b.WriteString(text[from:])
	return b.String()
}

// homeStandsAsPath reports whether text[at:end] — an occurrence of the home —
// is a path of its own rather than part of another. The trailing half: the
// name does not continue past it (so "/rootfs" is not "/root", while
// "/root/x", "/root" at the end and "/root-cause" are — see nameContinues).
// The leading half applies to a SINGLE-segment home only: "/var/root" and
// "~/root" are not "/root" (a preceding '/' is an empty segment, as in
// "file:///root", and does not count). A home of two or more segments
// ("/Users/me", "/home/me") carries the caller's name inside it, and a longer abcd-audit:allow
// root before it — a backup volume, a container mount — does not make that
// name someone else's, so it is swept wherever it sits; the leading anchor's
// one purpose, telling "/root" from "/var/root", has no counterpart there.
func homeStandsAsPath(text string, at, end int) bool {
	single := strings.IndexByte(text[at+1:end], '/') < 0
	if single && at > 0 && text[at-1] != '/' && (isPathSegmentByte(text[at-1]) || text[at-1] == '~') {
		return false
	}
	return !nameContinues(text, end)
}

// homeSweepable is homeStandsAsPath with the leading half of the anchor
// waived inside a URL span: behind a URL host the byte before the home is the
// host's last letter, which is a path-segment byte, yet the path IS the
// caller's home ("https://ci.example.com/Users/me/build.log"). The trailing abcd-audit:allow
// half still holds there, so a longer name behind a host is not swept either.
func homeSweepable(text string, at, end int, urls []span) bool {
	if inAnySpan(at, urls) {
		return !nameContinues(text, end)
	}
	return homeStandsAsPath(text, at, end)
}

// nameContinues is the ONE rule the home-path anchor uses for "the name goes
// on", shared by the detector, SweepCallerHome, the backstop and the
// home_path_other skip: a letter or digit continues it, and nothing else does.
// The only false positive the trailing anchor exists for is a longer
// alphanumeric name that starts with the home ("/Users/alexandra" under abcd-audit:allow
// HOME=/Users/alex, "/rootfs" under HOME=/root). '.', '-' and '_' are abcd-audit:allow
// boundaries: "/Users/me.zip", "/Users/me-old" and "/Users/me_snapshot" are abcd-audit:allow
// the caller's name with a suffix, not another user, and treating the
// punctuation as a continuation let every one of them through the detector
// and the backstop alike. Over-sweeping "/root-cause" to "~-cause" is the
// safe side; the unanchored sweep did the same.
//
// This is deliberately NOT wordBounded's rule: local_username matches the
// bare username as a word, where '_' must continue the word so "me" does not
// fire inside "me_2"; the home path is a longer literal that carries its own
// separators, so a suffix after it is a boundary here.
func nameContinues(text string, end int) bool {
	return end < len(text) && isAlnumByte(text[end])
}

func isAlnumByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// SurvivingCallerHome is the deterministic backstop after SweepCallerHome,
// with no dependency on the pattern heuristic. It REWRITES what it can and
// reports what it cannot: a "/Users/<user>" / "/home/<user>" segment for the
// caller's local username (basename of $HOME) is rewritten to the
// local_username placeholder wherever the name does not go on — whatever
// punctuation follows it, and whatever precedes it, since "/Users/me" behind a abcd-audit:allow
// URL host or under a longer root is still the caller's name — and the $HOME
// literal standing as a path is reported (defensive: the sweep removes exactly
// those, so this fires only if the two ever disagree). A backstop that refused
// the write on a segment it could rewrite stopped every page that quoted the
// caller's own path; one that rewrites keeps the write and drops the name.
// Returned findings carry only the kind (masked Matched), enough for a
// refusal to report without exposing raw material.
func SurvivingCallerHome(text, home string) (string, []Finding) {
	var out []Finding
	if home == "" {
		return text, nil
	}
	user := home
	if i := strings.LastIndex(home, "/"); i >= 0 {
		user = home[i+1:]
	}
	if user != "" {
		for _, prefix := range []string{"/Users/", "/home/"} {
			text = sweepUserSegment(text, prefix+user, prefix+redactionReplacement(Finding{Kind: kindLocalUser}))
		}
	}
	if SweepCallerHome(text, home) != text {
		out = append(out, Finding{Kind: kindHomeSelf, Matched: "~"})
	}
	return text, out
}

// sweepUserSegment rewrites every occurrence of needle ("/Users/<user>" or
// "/home/<user>") that stands as a complete path segment, by the trailing
// half of the sweep's anchor (nameContinues): "/Users/me" does not falsely abcd-audit:allow
// match "/Users/metoo" (a different, longer username), while "/Users/me." at abcd-audit:allow
// a sentence end does. Only the trailing half — a leading anchor here would
// trade the old refusal for a leak, since a name behind a host or under a
// longer root is still the caller's name.
func sweepUserSegment(text, needle, repl string) string {
	var b strings.Builder
	from := 0
	for {
		i := strings.Index(text[from:], needle)
		if i < 0 {
			break
		}
		at := from + i
		end := at + len(needle)
		if !nameContinues(text, end) {
			b.WriteString(text[from:at])
			b.WriteString(repl)
			from = end
			continue
		}
		b.WriteString(text[from : at+1])
		from = at + 1
	}
	if from == 0 {
		return text
	}
	b.WriteString(text[from:])
	return b.String()
}
