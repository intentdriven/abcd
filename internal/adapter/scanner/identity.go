package scanner

import (
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// Identity is the caller's runtime identity, probed from git config and the
// environment. Its matchers are built at scan time; empty fields disable the
// corresponding kind.
type Identity struct {
	GitUserName  string
	GitUserEmail string
	// OtherGitUserNames and OtherGitUserEmails are the caller's OTHER git
	// identities: every user.name / user.email value git resolves for this
	// repository in a scope the effective value displaced — the unconditional
	// global identity under a repo-local persona, or the reverse, or an
	// includeIf persona keyed on where the repository sits — and the
	// GIT_AUTHOR_*/GIT_COMMITTER_* persona the environment sets, which no
	// config listing reports at all. A persona ADDS an identity to redact; it
	// never replaces one (GHSA-v826-5jf4-p8xg, GHSA-gxhr-pmwv-r99p,
	// GHSA-rvhr-3455-c5jw).
	OtherGitUserNames  []string
	OtherGitUserEmails []string
	GitRemoteUsername  string
	HomePath           string
	HomeUser           string
}

// Built-in identity kinds.
const (
	kindHomeSelf   = "home_path_self"
	kindHomeOther  = "home_path_other"
	kindRealEmail  = "real_email"
	kindRealName   = "real_name"
	kindGithubUser = "github_username"
	kindLocalUser  = "local_username"
)

// DefaultIdentitySeverities is the built-in severity floor per identity kind
// (ported from pii.py DEFAULT_IDENTITY_SEVERITIES). A config override may raise
// but never lower these.
func DefaultIdentitySeverities() map[string]Severity {
	return map[string]Severity{
		kindHomeSelf:   SeverityHardFail,
		kindHomeOther:  SeverityWarn,
		kindRealEmail:  SeverityHardFail,
		kindRealName:   SeverityHardFail,
		kindGithubUser: SeverityWarn,
		kindLocalUser:  SeverityHardFail,
	}
}

// ProbeIdentity gathers the caller's identity from git config and $HOME,
// best-effort: any probe that fails leaves its field empty. repoRoot scopes the
// git config reads so a per-repo user.name/email is honoured — and honoured in
// ADDITION to the caller's other identities, not instead of them: the name and
// email fields hold the value git resolves in this repository, and the Other*
// fields hold every value another scope configured that it displaced.
func ProbeIdentity(repoRoot string) Identity {
	var id Identity
	git := func(args ...string) string {
		full := append([]string{"-C", repoRoot}, args...)
		cmd := exec.Command("git", full...)
		// Scrub repo-selection and config-injection env vars, but keep global
		// config: this probe reads the caller's OWN user.name/user.email to redact
		// their identity, and those live in global config, so full IsolatedEnv
		// (which neutralises ~/.gitconfig) would blind the identity gate. Scrubbing
		// still stops an inherited GIT_DIR pointing the probe at another repo and an
		// injected GIT_CONFIG_* forging a fake identity that displaces the real one.
		cmd.Env = gitutil.ScrubbedEnv()
		out, err := cmd.Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	// --get-all lists every value git resolves for the key, in scope order
	// with the effective one last — system, global with its includeIf
	// includes evaluated where they sit, repo-local, worktree. --get returned
	// only that last value, so a repo-local or includeIf persona displaced the
	// caller's global identity from the matcher set and the displaced identity
	// was stored in clear text. Neither --local nor --global sees an includeIf
	// persona for what it is (the former misses it, the latter hides it behind
	// the unconditional value), which is why the union comes from ONE
	// unscoped listing rather than a scope-by-scope reassembly.
	id.GitUserName, id.OtherGitUserNames = splitIdentityValues(git("config", "--get-all", "user.name"))
	id.GitUserEmail, id.OtherGitUserEmails = splitIdentityValues(git("config", "--get-all", "user.email"))
	// GIT_AUTHOR_* and GIT_COMMITTER_* are an identity scope `git config` never
	// reports and that outranks every config file when a commit is written: a CI
	// runner, a direnv profile and a rebase wrapper all set them. The persona
	// that AUTHORS the caller's commits was therefore absent from the matcher
	// set and stored in clear. They are read from the process environment (not
	// through the scrubbed subprocess env, which deliberately does not carry
	// them) and folded in as OTHERS: an injected value can only ADD something to
	// redact, never displace the identity the config resolves, so the
	// config-injection guard above is not weakened by reading them.
	id.OtherGitUserNames = addIdentityValues(id.GitUserName, id.OtherGitUserNames,
		os.Getenv("GIT_AUTHOR_NAME"), os.Getenv("GIT_COMMITTER_NAME"))
	id.OtherGitUserEmails = addIdentityValues(id.GitUserEmail, id.OtherGitUserEmails,
		os.Getenv("GIT_AUTHOR_EMAIL"), os.Getenv("GIT_COMMITTER_EMAIL"))
	if remote := git("config", "--get", "remote.origin.url"); remote != "" {
		if m := githubRemoteRe.FindStringSubmatch(remote); m != nil {
			id.GitRemoteUsername = m[1]
		}
	}
	if home := CallerHome(); home != "" {
		id.HomePath = home
		if i := strings.LastIndex(id.HomePath, "/"); i >= 0 {
			id.HomeUser = id.HomePath[i+1:]
		}
	}
	return id
}

// splitIdentityValues turns a `git config --get-all` listing into the
// effective (last) value and the distinct others it displaced — trimmed,
// empties dropped, and de-duplicated case-insensitively, the way every
// identity matcher compares.
func splitIdentityValues(listing string) (effective string, others []string) {
	var vals []string
	for _, v := range strings.Split(listing, "\n") {
		if v = strings.TrimSpace(v); v != "" {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		return "", nil
	}
	effective = vals[len(vals)-1]
	for _, v := range vals[:len(vals)-1] {
		if strings.EqualFold(v, effective) || containsFold(others, v) {
			continue
		}
		others = append(others, v)
	}
	return effective, others
}

// addIdentityValues folds extra values into an Other* set under the same guards
// splitIdentityValues applies: trimmed, empties dropped, and dropped again when
// they only repeat the effective value or one already in the set.
func addIdentityValues(effective string, others []string, extra ...string) []string {
	for _, v := range extra {
		if v = strings.TrimSpace(v); v == "" {
			continue
		}
		if strings.EqualFold(v, effective) || containsFold(others, v) {
			continue
		}
		others = append(others, v)
	}
	return others
}

// identityValues lists one identity field's values to match — the effective
// value and the others — trimmed, non-empty, de-duplicated case-insensitively,
// and longest first, so an alternation built from them never settles for a
// shorter value that is a prefix of a longer one at the same offset.
func identityValues(effective string, others []string) []string {
	var out []string
	for _, v := range append([]string{effective}, others...) {
		v = strings.TrimSpace(v)
		if v == "" || containsFold(out, v) {
			continue
		}
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}

// minEmailRunes is the shortest value the email matcher will arm on. The
// shortest address anyone actually holds is a@b.c; anything below that is a
// placeholder or a fragment, and an alternation is only as safe as its
// shortest branch.
const minEmailRunes = 5

// plausibleEmail reports whether a configured value is shaped like an address:
// an '@' with a non-empty local part before it and a non-empty domain after,
// and long enough that matching it literally cannot sweep ordinary prose.
func plausibleEmail(v string) bool {
	if utf8.RuneCountInString(v) < minEmailRunes {
		return false
	}
	at := strings.IndexByte(v, '@')
	return at > 0 && at < len(v)-1
}

func containsFold(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(x, v) {
			return true
		}
	}
	return false
}

// foldedAlternation compiles values into one case-insensitive literal
// alternation. The (?i) is what every identity matcher already carried; the
// alternation is what lets one matcher stand for every scope's value.
func foldedAlternation(values []string) *regexp.Regexp {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = regexp.QuoteMeta(v)
	}
	return regexp.MustCompile(`(?i)(?:` + strings.Join(quoted, `|`) + `)`)
}

// isPublicHandle reports whether a git user.name is the caller's public GitHub
// handle rather than a real name: equal to the remote's owner, or — for the
// effective name alone — to the login the effective noreply address carries.
// The remote-owner comparison
// alone breaks on an org-owned remote (iss-283): the owner stops being the
// caller the moment the repo transfers, and the caller's public handle would
// start scanning as a real name. The noreply address carries the caller's own
// GitHub login locally, so a user.name equal to that login is the same public
// handle, whatever the remote's owner is. A handle is reported as
// github_username, never promoted to a hard-fail real_name.
func isPublicHandle(name string, id Identity) bool {
	if id.GitRemoteUsername != "" && strings.EqualFold(name, id.GitRemoteUsername) {
		return true
	}
	// The noreply provision is confined to the EFFECTIVE pair — this
	// repository's user.name against this repository's user.email — because
	// that pairing is the only one git asserts. Read across the scope union it
	// becomes a way to DISARM redaction: a global noreply address makes a
	// repo-local user.name equal to its login a "public handle", and on an
	// org-owned remote the github_username matcher (the remote's owner alone)
	// does not cover that name either, so nothing redacts a name the
	// single-identity probe hard-failed on. Widening the scopes must never
	// SUBTRACT a value from real_name.
	if !strings.EqualFold(name, id.GitUserName) {
		return false
	}
	lm := noreplyLoginRe.FindStringSubmatch(id.GitUserEmail)
	return lm != nil && strings.EqualFold(name, lm[1])
}

var (
	// GitHub username inside a remote URL (https or ssh form). Case-insensitive
	// on the host: git stores the remote verbatim, so a hand-typed GitHub.com
	// must still resolve the handle, or github_username redaction never arms.
	githubRemoteRe = regexp.MustCompile(`(?i)github\.com[:/]([A-Za-z0-9-]+)/`)
	// Generic home path. Both boundaries are Go predicates: a leading RE2 \b is
	// wrong here — it is an ASCII word boundary that requires a WORD character
	// immediately before the '/', which never holds at line start or after a
	// space/'='/quote, so it silently killed home_path_other detection for every
	// realistic occurrence. leadingBoundaryOK enforces the real requirement (the
	// '/' does not continue a longer path segment); the trailing boundary stays
	// trailingBoundaryOK.
	genericHomeRe = regexp.MustCompile(`(?:/Users/[A-Za-z0-9._-]+|/home/[A-Za-z0-9._-]+)`)
	// Loose URL span (scheme to whitespace/quote/closing).
	urlSpanRe = regexp.MustCompile(`(?:https?://|git@|ftp://|ssh://)[^\s"'` + "`" + `)>\]<]+`)
	// A git noreply email is not a leak.
	noreplyRe = regexp.MustCompile(`(?i)@users\.noreply\.github\.com$`)
	// The GitHub login embedded in a users.noreply.github.com address, in both
	// its forms ("id+login@..." and the legacy "login@...").
	noreplyLoginRe = regexp.MustCompile(`(?i)^(?:[0-9]+\+)?([A-Za-z0-9-]+)@users\.noreply\.github\.com$`)
)

// homeBoundary is the trailing-boundary set for a home-path match (ported from
// the Python lookahead [/\s"'`)\]\}<,;:]).
func homeBoundary(r rune) bool {
	switch r {
	case '/', '"', '\'', '`', ')', ']', '}', '<', ',', ';', ':':
		return true
	}
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' || r == '\v'
}

// identityMatchers holds the per-scan compiled identity regexes.
type identityMatchers struct {
	id Identity
	// bytes says the line comes from a raw blob rather than text. A blob has
	// no path syntax to anchor on — the byte before the caller's home is
	// whatever the format put there — so the leading half of the home anchor
	// is waived on bytes; the literal is long enough that a chance collision
	// is negligible, and the byte policy drops local_username, which is the
	// rule that would otherwise have caught the name (iss-2608292034215745).
	bytes        bool
	homeSelf     *regexp.Regexp
	email        *regexp.Regexp // every scope's user.email, one alternation
	name         *regexp.Regexp // every scope's user.name that is not a public handle
	github       *regexp.Regexp
	localBare    *regexp.Regexp
	localEncoded string // path-encoded username (dots->hyphens); boundary checked in Go
}

func newIdentityMatchers(id Identity) identityMatchers {
	m := identityMatchers{id: id}
	if id.HomePath != "" {
		// Case-insensitive: on a case-folding filesystem (macOS/Windows) a differently
		// cased spelling of the caller's own home path resolves to the SAME directory,
		// so a case variant must still trip the hard_fail home_path_self gate — matching
		// the (?i) already applied to the email/name/github matchers below.
		m.homeSelf = regexp.MustCompile(`(?i)` + regexp.QuoteMeta(id.HomePath))
	}
	var emails []string
	for _, e := range identityValues(id.GitUserEmail, id.OtherGitUserEmails) {
		// A value has to look like an address before it arms the matcher. The
		// name matcher has always dropped a value under three runes; the email
		// matcher dropped nothing, so a one-letter placeholder from any scope
		// — a stub in a CI config, a fragment left by splitting a value with an
		// embedded newline — compiled into the alternation and every "e" in the
		// text then scanned as the caller's hard_fail real_email, redacting the
		// prose it appeared in.
		if plausibleEmail(e) {
			emails = append(emails, e)
		}
	}
	if len(emails) > 0 {
		// Case-insensitive: email addresses are compared case-insensitively in
		// practice (the domain always, and mailbox providers overwhelmingly), so a
		// trivial case variant of the caller's own address must not slip the
		// hard_fail real_email gate.
		m.email = foldedAlternation(emails)
	}
	var names []string
	for _, n := range identityValues(id.GitUserName, id.OtherGitUserNames) {
		// A public handle is reported as github_username by its own matcher,
		// never as a hard-fail real_name; a name under three runes is too
		// short to be one.
		if len(n) >= 3 && !isPublicHandle(n, id) {
			names = append(names, n)
		}
	}
	if len(names) > 0 {
		// No RE2 \b: it is ASCII-only, so a name whose first or last rune is
		// non-ASCII (accented, CJK, Cyrillic) never satisfies the boundary and the
		// hard_fail real_name detector silently never fires. The word boundary is a
		// Unicode-aware Go predicate applied to each match instead.
		m.name = foldedAlternation(names)
	}
	if id.GitRemoteUsername != "" {
		// GitHub usernames are case-insensitive; \b dropped for the same
		// Unicode-boundary reason as real_name (see above).
		m.github = regexp.MustCompile(`(?i)` + regexp.QuoteMeta(id.GitRemoteUsername))
	}
	if id.HomeUser != "" {
		// Case-insensitive, for the same reason homeSelf is (above): HomeUser is
		// the last segment of that same HomePath, so on a case-folding filesystem
		// a case variant of the login (the natural prose spelling in a transcript)
		// resolves to the same account and must still trip the hard_fail
		// local_username gate — not slip redaction while the home path is caught.
		m.localBare = regexp.MustCompile(`(?i)` + regexp.QuoteMeta(id.HomeUser))
		if enc := strings.ReplaceAll(id.HomeUser, ".", "-"); enc != id.HomeUser {
			m.localEncoded = enc
		}
	}
	return m
}

// span is a half-open byte interval on a line.
type span struct{ start, end int }

func inAnySpan(pos int, spans []span) bool {
	for _, s := range spans {
		if s.start <= pos && pos < s.end {
			return true
		}
	}
	return false
}

// urlSpans returns the URL-like spans on a line.
func urlSpans(line string) []span {
	var out []span
	for _, loc := range urlSpanRe.FindAllStringIndex(line, -1) {
		out = append(out, span{loc[0], loc[1]})
	}
	return out
}

// findings scans one line for identity-derived matches, applying every ported
// suppression, and returns findings tagged with the merged identity severities.
func (m identityMatchers) findings(line string, lineno int, id2sev map[string]Severity, file string) []Finding {
	var out []Finding
	sevFor := func(kind string) Severity {
		if s, ok := id2sev[kind]; ok {
			return s
		}
		return defaultPatternSeverity
	}
	add := func(kind string, col int, matched, suggested string) {
		out = append(out, Finding{
			File: file, Line: lineno, Column: col, Kind: kind,
			Severity: sevFor(kind), Snippet: snippet(line), Matched: matched,
			Suggested: suggested, line: line,
		})
	}

	urls := urlSpans(line)

	// home_path_self — the caller's OWN home path (hard_fail). Detected
	// regardless of the trailing rune: the trailing-boundary heuristic exists
	// only to avoid over-flagging a DIFFERENT user's path (home_path_other),
	// never to license leaving the caller's own home path unredacted. A home
	// path followed by punctuation (e.g. "/Users/me#draft", "$HOME/dir&") is abcd-audit:allow
	// still the caller's home and must be redacted. What is NOT the caller's
	// home is a longer NAME that merely starts with it — "/rootfs/etc/hosts"
	// under HOME=/root, "/home/abc" under HOME=/home/a — so a match must stand
	// as a path of its own, by the same anchor SweepCallerHome applies; the
	// suppression spans below are filtered by it too, so a dropped span does
	// not go on hiding the local_username underneath it. Inside a URL the
	// byte before the home is the host's last letter, so the leading half of
	// the anchor is waived there (homeSweepable): a home behind a URL host is
	// still the caller's home, and no other detector reaches it — local_username
	// is URL-suppressed and home_path_other stops at the same byte.
	if m.homeSelf != nil {
		for _, loc := range m.homeSelf.FindAllStringIndex(line, -1) {
			stands := homeSweepable(line, loc[0], loc[1], urls)
			if m.bytes {
				// A raw blob has no path syntax on either side of the
				// literal, so neither half of the anchor applies: the
				// long home literal is its own evidence there.
				stands = true
			}
			if !stands {
				continue
			}
			add(kindHomeSelf, loc[0]+1, line[loc[0]:loc[1]], "~")
		}
	}
	// home_path_other — a generic /Users|/home path that is not the caller's own.
	for _, loc := range genericHomeRe.FindAllStringIndex(line, -1) {
		if !leadingBoundaryOK(line, loc[0]) || !trailingBoundaryOK(line, loc[1]) {
			continue
		}
		matched := line[loc[0]:loc[1]]
		if m.homeSelf != nil && homeSelfStandsIn(m.homeSelf, matched) {
			continue
		}
		// /Users/Shared and friends are macOS system directories, not users
		// (iss-153). The audit rule applies the same allowlist, so the two
		// detectors cannot disagree about what a username is.
		//
		// The exemption stops at the system directory ITSELF: a name nested
		// under it (/Users/Shared/<user>/...) is still a user, and letting the
		// one-segment match end on the exempt segment made the system directory
		// a shield. When a further segment follows, the match is EXTENDED over
		// it so the redacted span covers the name, not just the prefix.
		if isNonUserHomeMatch(matched) {
			end, ok := nextPathSegmentEnd(line, loc[1])
			if !ok {
				continue // the system directory alone, or with no further segment
			}
			matched = line[loc[0]:end]
		}
		add(kindHomeOther, loc[0]+1, matched, "(remove or relativise — third-party path)")
	}
	// real_email — skip the noreply form.
	if m.email != nil {
		for _, loc := range m.email.FindAllStringIndex(line, -1) {
			matched := line[loc[0]:loc[1]]
			if noreplyRe.MatchString(matched) {
				continue
			}
			add(kindRealEmail, loc[0]+1, matched, "<github-userid>@users.noreply.github.com or remove")
		}
	}
	// real_name — suppress inside URL spans (a name that is the public handle
	// was left out of the matcher: isPublicHandle).
	if m.name != nil {
		for _, loc := range m.name.FindAllStringIndex(line, -1) {
			if !wordBounded(line, loc[0], loc[1]) {
				continue
			}
			if inAnySpan(loc[0], urls) {
				continue
			}
			add(kindRealName, loc[0]+1, line[loc[0]:loc[1]], "(remove or replace with persona)")
		}
	}
	// github_username — suppress inside URL spans.
	if m.github != nil {
		for _, loc := range m.github.FindAllStringIndex(line, -1) {
			if !wordBounded(line, loc[0], loc[1]) {
				continue
			}
			if inAnySpan(loc[0], urls) {
				continue
			}
			add(kindGithubUser, loc[0]+1, line[loc[0]:loc[1]], "(review — may be intentional in repo URL contexts)")
		}
	}
	// local_username — suppress inside home/generic-home/email/URL spans.
	if m.localBare != nil {
		supp := m.localSuppressionSpans(line, urls)
		emit := func(loc []int) {
			if inAnySpan(loc[0], supp) {
				return
			}
			// A username that equals a system directory and appears as the top
			// segment of an absolute path (e.g. "/dev/null" when the machine user
			// is "dev") is a system path, not an identity leak.
			if isSystemPathSegment(line, loc[0], loc[1]) {
				return
			}
			add(kindLocalUser, loc[0]+1, line[loc[0]:loc[1]],
				"(local machine username; replace with [USERNAME] or remove)")
		}
		for _, loc := range m.localBare.FindAllStringIndex(line, -1) {
			if !wordBounded(line, loc[0], loc[1]) {
				continue
			}
			emit(loc)
		}
		if m.localEncoded != "" {
			for _, loc := range encodedMatches(line, m.localEncoded) {
				emit(loc)
			}
		}
	}
	return out
}

// isNonUserHomeMatch reports whether a generic-home match's final segment is a
// well-known non-user directory under a /Users root.
func isNonUserHomeMatch(matched string) bool {
	if !strings.HasPrefix(strings.ToLower(matched), "/users/") {
		return false
	}
	i := strings.LastIndexByte(matched, '/')
	return i >= 0 && IsNonUserHomeSegment(matched[i+1:])
}

// nextPathSegmentEnd returns the end offset of the NAME-BEARING path segment
// that follows pos, and whether one is there at all. "/Users/Shared" and
// "/Users/Shared/" have none; nor does a segment of pure dots on its own, which
// is prose ("/Users/Shared/...") or a relative marker, never a username.
// "/Users/Shared/<name>/x" has "<name>".
//
// A dots-only or an empty segment does not end the walk, though: either one
// between the system directory and a name ("/Users/Shared/../<user>",
// "/Users/Shared//<user>") would otherwise re-create the shield the exemption is
// not allowed to give. genericHomeRe is POSIX-only, so '/' is the only separator
// that can reach here.
func nextPathSegmentEnd(line string, pos int) (int, bool) {
	for pos < len(line) && line[pos] == '/' {
		i, named := pos+1, false
		for i < len(line) && isHomeSegmentByte(line[i]) {
			if line[i] != '.' {
				named = true
			}
			i++
		}
		if named {
			return i, true
		}
		pos = i // an empty or dots-only segment: skip it and keep looking
	}
	return 0, false
}

// isHomeSegmentByte matches the character class genericHomeRe uses for a
// username segment, so nextPathSegmentEnd walks exactly the span the regex
// would match. It is the regex's class, not the home-path anchor's: the
// anchor (nameContinues) treats '.', '_' and '-' as boundaries because a
// suffix after the caller's home is still the caller's name.
func isHomeSegmentByte(b byte) bool {
	return b == '.' || b == '_' || b == '-' ||
		(b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// homeSelfStandsIn reports whether the caller's home occurs in matched as a
// path of its own, by the anchor the home_path_self detector applies. A
// generic-home match whose name merely starts with the home basename
// ("/Users/alexandra" under HOME=/Users/alex) is a DIFFERENT user's path: the abcd-audit:allow
// anchored detector declines it as the caller's own, so the home_path_other
// skip must decline it too, or nothing reports it at all.
func homeSelfStandsIn(homeSelf *regexp.Regexp, matched string) bool {
	for _, loc := range homeSelf.FindAllStringIndex(matched, -1) {
		if homeStandsAsPath(matched, loc[0], loc[1]) {
			return true
		}
	}
	return false
}

// localSuppressionSpans returns spans where a local-username match is not a
// standalone leak: the caller's own home path (home_path_self, always redacted
// hard_fail), the exact email, and URLs. home_path_other spans are deliberately
// NOT included: home_path_other is only a WARN, so suppressing a hard_fail
// local_username underneath one would downgrade a username leak (e.g. the
// "<user>" in "/home/<user>/...") out of the ship-blocking gate. Letting both
// findings fire keeps the hard_fail signal and still redacts the span.
func (m identityMatchers) localSuppressionSpans(line string, urls []span) []span {
	spans := append([]span(nil), urls...)
	if m.homeSelf != nil {
		for _, loc := range m.homeSelf.FindAllStringIndex(line, -1) {
			if !homeSweepable(line, loc[0], loc[1], urls) {
				continue // not reported as the home, so it must not suppress the username either
			}
			spans = append(spans, span{loc[0], loc[1]})
		}
	}
	if m.email != nil {
		for _, loc := range m.email.FindAllStringIndex(line, -1) {
			spans = append(spans, span{loc[0], loc[1]})
		}
	}
	return spans
}

// encodedMatches finds the path-encoded username with the ported custom
// boundary: preceded by start-of-string or a non-[A-Za-z0-9.] rune (the RE2
// lookbehind replacement) and followed by EOL or a non-[A-Za-z0-9.] rune.
func encodedMatches(line, encoded string) [][]int {
	var out [][]int
	// Case-insensitive, matching the folded m.localBare matcher: the encoded
	// (dot->dash) spelling of the login must be redacted whatever its case. The
	// window is compared with EqualFold rather than lower-casing the whole line,
	// whose byte length can shift on non-ASCII input and corrupt the offsets.
	for start := 0; start+len(encoded) <= len(line); start++ {
		end := start + len(encoded)
		if !strings.EqualFold(line[start:end], encoded) {
			continue
		}
		if boundaryBefore(line, start) && boundaryAfter(line, end) {
			out = append(out, []int{start, end})
		}
	}
	return out
}

func isUsernameWordRune(r byte) bool {
	return r == '.' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// systemDirNames are well-known absolute top-level system directories. A local
// username equal to one of these that appears as the first segment of an
// absolute path is a system path, not an identity leak — a genuine username
// leak is nested under a home root (/Users/<u>, /home/<u>), never at the
// filesystem root. Suppressing only this exact collision (iss-31: "/dev/null"
// when the machine user is "dev") keeps genuine leak detection intact.
var systemDirNames = map[string]bool{
	"dev": true, "proc": true, "sys": true, "usr": true, "bin": true,
	"sbin": true, "etc": true, "var": true, "tmp": true, "opt": true,
	"lib": true, "run": true, "boot": true, "mnt": true, "media": true,
	"srv": true, "root": true,
}

// isSystemPathSegment reports whether line[start:end] is the first segment of an
// absolute Unix path naming a well-known system directory (e.g. the "dev" in
// "/dev/null"). It requires a leading root '/' that is not itself nested under a
// prior path segment, and a trailing '/', so "/Users/<user>/x" and a bare "dev"
// are NOT suppressed.
func isSystemPathSegment(line string, start, end int) bool {
	if !systemDirNames[strings.ToLower(line[start:end])] {
		return false
	}
	if end >= len(line) || line[end] != '/' {
		return false
	}
	if start == 0 || line[start-1] != '/' {
		return false
	}
	root := start - 1
	return root == 0 || !isPathSegmentByte(line[root-1])
}

// isPathSegmentByte reports whether b can be part of a path segment, used to
// decide whether a '/' begins an absolute path or continues a nested one.
func isPathSegmentByte(b byte) bool {
	return b == '/' || b == '.' || b == '-' || b == '_' ||
		(b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

func boundaryBefore(line string, pos int) bool {
	if pos == 0 {
		return true
	}
	return !isUsernameWordRune(line[pos-1])
}

func boundaryAfter(line string, pos int) bool {
	if pos >= len(line) {
		return true
	}
	return !isUsernameWordRune(line[pos])
}

// trailingBoundaryOK reports whether the rune at byte offset end is a home-path
// boundary or the line ends there.
func trailingBoundaryOK(line string, end int) bool {
	if end >= len(line) {
		return true
	}
	return homeBoundary(rune(line[end]))
}

// leadingBoundaryOK reports whether byte offset start begins a home path rather
// than continuing a longer path segment: it is the line start, or the preceding
// byte is not a path-segment byte. This replaces the broken leading RE2 \b on
// genericHomeRe.
func leadingBoundaryOK(line string, start int) bool {
	if start == 0 {
		return true
	}
	return !isPathSegmentByte(line[start-1])
}

// isWordRune reports whether r is a Unicode word rune (letter, digit, or '_') —
// the class RE2's ASCII-only \b cannot see for non-ASCII letters. It bounds
// the bare-token matchers (local_username, github_username, real_name), where
// '_' continues a word so "me" does not fire inside "me_2"; the home-path
// anchor uses nameContinues instead, where '_' is a boundary, because a
// suffix after the caller's home is still the caller's home.
func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// wordBoundaryAt reports whether a Unicode word boundary falls at byte offset pos
// in line: exactly one of the runes immediately before and at pos is a word rune
// (ends of the string count as non-word). It is the Unicode-aware stand-in for
// the \b assertions dropped from the identity matchers so accented/CJK/Cyrillic
// names and usernames are bounded correctly.
func wordBoundaryAt(line string, pos int) bool {
	beforeWord := false
	if pos > 0 {
		r, _ := utf8.DecodeLastRuneInString(line[:pos])
		beforeWord = isWordRune(r)
	}
	afterWord := false
	if pos < len(line) {
		r, _ := utf8.DecodeRuneInString(line[pos:])
		afterWord = isWordRune(r)
	}
	return beforeWord != afterWord
}

// wordBounded reports whether the half-open match [start,end) sits on Unicode
// word boundaries at both ends.
func wordBounded(line string, start, end int) bool {
	return wordBoundaryAt(line, start) && wordBoundaryAt(line, end)
}

// snippet is the trimmed line capped at 200 bytes.
func snippet(line string) string {
	s := strings.TrimSpace(line)
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
