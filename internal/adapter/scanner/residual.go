package scanner

import (
	"os"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
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
//
// The home is read in every spelling the detector reads (backstopSpans): as
// written, and through the decoded views of each line that carries an escape,
// so "\/Users\/me", its \u002f form, a second JSON layer and "%2FUsers%2Fme" abcd-lint:allow
// are swept with the escape's own bytes, exactly as the literal home is
// (iss-2609261659041553).
func SweepCallerHome(text, home string) string {
	if home == "" {
		return text
	}
	return replaceSpans(text, backstopSpans(text, home, true, homeSweepable), "~")
}

// backstopSpans returns, sorted and disjoint, the byte spans of text at which
// needle occurs and accept holds, read in the spellings the detector reads: as
// written, and through the decoded views of every line carrying a backslash or
// a '%' (the percent pre-pass's view and each JSON-escape layer, the one
// definition in percent.go and jsonescape.go). A hit on a view is mapped back
// through the view's position map, so the span covers the whole escape units
// it was decoded from and a rewrite never splits one. accept judges an
// occurrence on the text it was found in, so the anchors read the decoded
// neighbours of an escaped home rather than the escape's letters; wantURLs
// says whether it consults the URL spans, which are then found once per text.
//
// An occurrence written straight after an odd run of backslashes is the tail
// of an escape ("\/Users/me" read from its '/'), so the raw reading leaves it abcd-lint:allow
// to the view that decodes the escape: judging the tail as written read the
// escape's backslash as a boundary and swept "x\/root" under HOME=/root, and
// rewriting from the '/' left a dangling "\~" no JSON reader accepts.
func backstopSpans(text, needle string, wantURLs bool, accept func(s string, at, end int, urls urlSet) bool) []span {
	out := needleOccurrences(text, needle, true, wantURLs, accept)
	for start := 0; start < len(text); {
		end := strings.IndexByte(text[start:], '\n')
		if end < 0 {
			end = len(text)
		} else {
			end += start
		}
		line := text[start:end]
		if strings.IndexByte(line, '\\') >= 0 || strings.IndexByte(line, '%') >= 0 {
			for _, v := range lineViews(line) {
				for _, sp := range needleOccurrences(v.text, needle, false, wantURLs, accept) {
					if rs, re, ok := mapDecodedSpan(v.posMap, sp.start, sp.end, len(line)); ok {
						out = append(out, span{start + rs, start + re})
					}
				}
			}
		}
		start = end + 1
	}
	return disjointSpans(out)
}

// needleOccurrences walks s for needle, keeping each occurrence accept holds
// and resuming past it, or one byte on where accept declines — the walk the
// sweep has always made. raw marks s as the text as written, where an
// occurrence behind an escape's backslash is left to the views.
func needleOccurrences(s, needle string, raw, wantURLs bool, accept func(s string, at, end int, urls urlSet) bool) []span {
	if needle == "" || !strings.Contains(s, needle) {
		return nil
	}
	var urls urlSet
	if wantURLs {
		urls = urlSpans(s)
	}
	var out []span
	from := 0
	for {
		i := strings.Index(s[from:], needle)
		if i < 0 {
			return out
		}
		at := from + i
		end := at + len(needle)
		if !(raw && afterEscapeBackslash(s, at)) && accept(s, at, end, urls) {
			out = append(out, span{at, end})
			from = end
			continue
		}
		from = at + 1
	}
}

// maxEscapeRunWalk bounds how far afterEscapeBackslash reads back. A run past
// it is judged even — the text as written decides, as it always did — so a
// crafted run of backslashes before every occurrence costs a constant each.
// Past the cap the raw reading spans the home alone, so the escape unit is
// still masked whole only because the JSON view of the same line spans the
// unit and disjointSpans unions the two readings: correctness there rests on
// that union, not on this walk.
const maxEscapeRunWalk = 64

// afterEscapeBackslash reports whether the byte at is escaped: an odd run of
// backslashes stands right before it.
func afterEscapeBackslash(s string, at int) bool {
	n := 0
	for j := at - 1; j >= 0 && s[j] == '\\' && n < maxEscapeRunWalk; j-- {
		n++
	}
	scanMeter.charge(stageIdentity, n)
	return n%2 == 1 && n < maxEscapeRunWalk
}

// disjointSpans sorts spans by start and unions the ones that overlap. Spans
// that merely touch stay apart, so two homes written back to back are two
// rewrites, as the literal sweep has always made them.
func disjointSpans(spans []span) []span {
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	scanMeter.charge(stageIdentity, len(spans)*searchCost(len(spans)))
	out := spans[:0]
	for _, s := range spans {
		if n := len(out); n > 0 && s.start < out[n-1].end {
			out[n-1].end = max(out[n-1].end, s.end)
			continue
		}
		out = append(out, s)
	}
	return out
}

// replaceSpans rewrites each of the sorted, disjoint spans of text to repl.
func replaceSpans(text string, spans []span, repl string) string {
	if len(spans) == 0 {
		return text
	}
	var b strings.Builder
	from := 0
	for _, sp := range spans {
		b.WriteString(text[from:sp.start])
		b.WriteString(repl)
		from = sp.end
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
	scanMeter.charge(stageIdentity, end-at)
	single := strings.IndexByte(text[at+1:end], '/') < 0
	if single && at > 0 && text[at-1] != '/' && (isPathSegmentByte(text[at-1]) || text[at-1] == '~') {
		return false
	}
	return !nameContinues(text, end)
}

// homeSweepable is homeStandsAsPath with the leading half of the anchor
// waived where the home is a URL's PATH ROOT: behind a URL host the byte
// before the home is the host's last letter, which is a path-segment byte, yet
// the path IS the caller's home ("https://ci.example.com/Users/me/build.log"). abcd-audit:allow
// The trailing half still holds there, so a longer name behind a host is not
// swept either. Deeper in a URL's path the ordinary anchor applies: the
// waiver used to hold anywhere inside the span, so under HOME=/root a
// "/root" segment anywhere in a URL path ("git@github.com:acme/root/tool.git",
// a module path ending in /root) was rewritten and hard-failed as the
// caller's home (iss-2608292005445725). A home of two or more segments is
// unaffected, since homeStandsAsPath never asks its leading anchor.
func homeSweepable(text string, at, end int, urls urlSet) bool {
	if atURLPathRoot(at, urls) {
		return !nameContinues(text, end)
	}
	return homeStandsAsPath(text, at, end)
}

// atURLPathRoot reports whether offset at is the first byte of the path of a
// URL span: the first '/' after "scheme://" and the authority, or the byte
// after the ':' of an scp-style "git@host:" remote (urlPathRoot).
func atURLPathRoot(at int, urls urlSet) bool {
	s := urls.at(at)
	return s != nil && s.root == at
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
//
// The rule itself lives in fsutil (NameContinues), so RedactRoot's trailing
// boundary and this one cannot drift apart (iss-2608292037564347).
func nameContinues(text string, end int) bool {
	return fsutil.NameContinues(text, end)
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
// longer root is still the caller's name. The segment is read in the same
// spellings as the home (backstopSpans), escaped ones included.
func sweepUserSegment(text, needle, repl string) string {
	return replaceSpans(text, backstopSpans(text, needle, false, func(s string, _, end int, _ urlSet) bool {
		return !nameContinues(s, end)
	}), repl)
}
