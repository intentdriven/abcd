package scanner

import (
	"math/bits"
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
	// GitRemoteRepo is the repository name of the same remote. With the owner
	// it spells the repository's own owner/repo slug, which is public by
	// construction and is not a github_username finding.
	GitRemoteRepo string
	HomePath      string
	HomeUser      string
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
		id.GitRemoteUsername, id.GitRemoteRepo = parseGitHubRemote(remote)
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
	githubRemoteRe = regexp.MustCompile(`(?i)github\.com[:/]([A-Za-z0-9-]+)/([A-Za-z0-9._-]*)`)
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

// parseGitHubRemote returns the owner and repository name of a GitHub remote
// URL (https, ssh or scp form), or two empty strings for any other remote. The
// owner is returned even where the repository name cannot be read.
func parseGitHubRemote(remote string) (owner, repo string) {
	m := githubRemoteRe.FindStringSubmatch(strings.TrimSpace(remote))
	if m == nil {
		return "", ""
	}
	return m[1], strings.TrimSuffix(m[2], ".git")
}

// isOwnRepoSlug reports whether the owner matched at line[start:end] is the
// owner half of the repository's own owner/repo slug: followed by '/', the
// repository name (case-insensitively, as the forge compares both), and a byte
// that cannot continue the name.
func isOwnRepoSlug(line string, end int, repo string) bool {
	if repo == "" || end >= len(line) || line[end] != '/' {
		return false
	}
	rest := line[end+1:]
	if len(rest) < len(repo) || !strings.EqualFold(rest[:len(repo)], repo) {
		return false
	}
	if len(rest) == len(repo) {
		return true
	}
	b := rest[len(repo)]
	return !(isAlnumByte(b) || b == '-' || b == '_')
}

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
	// localGeneric says the account name identifies no person — a role or
	// image default ("dev", "runner", "root") or a one- or two-rune name — so
	// the bare word is ordinary vocabulary and only an occurrence where an
	// account name stands is reported (isGenericAccountName, iss-236).
	localGeneric bool
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
		m.localGeneric = isGenericAccountName(id.HomeUser)
		if enc := strings.ReplaceAll(id.HomeUser, ".", "-"); enc != id.HomeUser {
			m.localEncoded = enc
		}
	}
	return m
}

// span is a half-open byte interval on a line.
type span struct{ start, end int }

// inAnySpan reports whether pos falls inside one of spans, which are sorted by
// start and disjoint (urlSet.spans, mergeSpans). The lookup is a binary search:
// a scan of the list for every match made a line dense in both matches and
// spans cost their product (iss-2609251535277823).
func inAnySpan(pos int, spans []span) bool {
	i := sort.Search(len(spans), func(i int) bool { return spans[i].end > pos })
	scanMeter.charge(stageIdentity, searchCost(len(spans)))
	return i < len(spans) && spans[i].start <= pos
}

// searchCost is what a binary search over n entries visits.
func searchCost(n int) int { return bits.Len(uint(n)) + 1 }

// mergeSpans sorts spans by start and merges the overlapping ones, which is the
// shape inAnySpan searches: the same positions, as a sorted disjoint list.
func mergeSpans(spans []span) []span {
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	scanMeter.charge(stageIdentity, len(spans)*searchCost(len(spans)))
	out := spans[:0]
	for _, s := range spans {
		if n := len(out); n > 0 && s.start <= out[n-1].end {
			if s.end > out[n-1].end {
				out[n-1].end = s.end
			}
			continue
		}
		out = append(out, s)
	}
	return out
}

// urlSpan is one URL-like span on a line with the offset its path begins at,
// or -1 where it has none: the first '/' after "scheme://" and the authority,
// or the byte after the ':' of an scp-style "git@host:" remote. The root is
// found once per span, so asking it of every match inside the span is a
// lookup rather than a search from the span's start (iss-2609251535277823).
type urlSpan struct {
	span
	root int
}

// urlSet is a line's URL spans. They are the leftmost non-overlapping matches
// of one regexp, so they come sorted by start and disjoint, and a position is
// found by binary search.
type urlSet []urlSpan

// urlSpans returns the URL-like spans on a line.
func urlSpans(line string) urlSet {
	scanMeter.charge(stageIdentity, len(line))
	var out urlSet
	for _, loc := range urlSpanRe.FindAllStringIndex(line, -1) {
		out = append(out, urlSpan{span{loc[0], loc[1]}, urlPathRoot(line, loc[0], loc[1])})
	}
	return out
}

// urlPathRoot returns the offset at which the URL line[start:end] begins its
// path, or -1.
func urlPathRoot(line string, start, end int) int {
	u := line[start:end]
	scanMeter.charge(stageIdentity, len(u))
	if i := strings.Index(u, "://"); i >= 0 {
		if p := strings.IndexByte(u[i+3:], '/'); p >= 0 {
			return start + i + 3 + p
		}
		return -1
	}
	if strings.HasPrefix(u, "git@") {
		if c := strings.IndexByte(u, ':'); c >= 0 {
			return start + c + 1
		}
	}
	return -1
}

// at returns the span containing pos, or nil.
func (u urlSet) at(pos int) *urlSpan {
	i := sort.Search(len(u), func(i int) bool { return u[i].end > pos })
	scanMeter.charge(stageIdentity, searchCost(len(u)))
	if i < len(u) && u[i].start <= pos {
		return &u[i]
	}
	return nil
}

// contains reports whether pos falls inside a URL span.
func (u urlSet) contains(pos int) bool { return u.at(pos) != nil }

// spans returns the bare intervals, sorted and disjoint.
func (u urlSet) spans() []span {
	out := make([]span, len(u))
	for i, s := range u {
		out[i] = s.span
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
		scanMeter.charge(stageIdentity, len(line))
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
	scanMeter.charge(stageIdentity, len(line))
	homeTokens := pathTokens{line: line}
	for _, loc := range genericHomeRe.FindAllStringIndex(line, -1) {
		if !(leadingBoundaryOK(line, loc[0]) || underAbsoluteRoot(line, loc[0], &homeTokens)) || !trailingBoundaryOK(line, loc[1]) {
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
		// The exemption covers the whole SUBTREE (iss-2609100505145554). A system
		// root is not a home root: the username position is the segment right
		// after /Users, and here it is held by a directory that names no user, so
		// nothing deeper is in that position either. The product creates such a
		// directory and names it in its own comments, tests and install docs, and
		// because home_path_other is an identity kind, BlockingResidual refused a
		// write on that text whatever its severity.
		//
		// It stops at a TRAVERSAL segment: "/Users/Shared/../<user>" and
		// "/Users/Shared//<user>" leave the shared root, so the name after them is
		// a home segment again and the match is EXTENDED over it so the redacted
		// span covers the name, not just the prefix. That is the half of the old
		// narrowing that was load-bearing.
		//
		// Deliberately NOT exempted here, unlike in the lint gate: a persona-named
		// home path. The gate judges curated committed text, where the roster is a
		// declaration that the name is fixture material; a transcript is live
		// session content, where a persona-shaped name is as likely to be a real
		// person, and the redactor's job is to fail safe.
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
		scanMeter.charge(stageIdentity, len(line))
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
		scanMeter.charge(stageIdentity, len(line))
		for _, loc := range m.name.FindAllStringIndex(line, -1) {
			if !wordBounded(line, loc[0], loc[1]) {
				continue
			}
			if urls.contains(loc[0]) {
				continue
			}
			add(kindRealName, loc[0]+1, line[loc[0]:loc[1]], "(remove or replace with persona)")
		}
	}
	// github_username — suppress inside URL spans.
	if m.github != nil {
		scanMeter.charge(stageIdentity, len(line))
		for _, loc := range m.github.FindAllStringIndex(line, -1) {
			if !wordBounded(line, loc[0], loc[1]) {
				continue
			}
			if urls.contains(loc[0]) {
				continue
			}
			if isOwnRepoSlug(line, loc[1], m.id.GitRemoteRepo) {
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
			// A whole component of a reverse-DNS identifier is a namespace, not a
			// home directory (iss-2609100505142469). Rewriting it corrupts the
			// technical content the record exists to hold, unrecoverably.
			if isDottedNamespaceComponent(line, loc[0], loc[1]) {
				return
			}
			// A generic account name is ordinary vocabulary wherever it does
			// not stand as an account (iss-236, iss-2609061504302157).
			if m.localGeneric && !standsAsAccountName(line, loc[0], loc[1], m.id.HomePath) {
				return
			}
			add(kindLocalUser, loc[0]+1, line[loc[0]:loc[1]],
				"(local machine username, the last segment of $HOME; replace with [USERNAME] or remove)")
		}
		scanMeter.charge(stageIdentity, len(line))
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

// genericAccountNames are account names that identify no person: the default
// accounts of hosted CI machines, container and cloud images and development
// environments, and the role words a shared machine is named for. Each is also
// ordinary vocabulary — a documented flag, a noun in the docs — so treating
// every bare occurrence as the caller's login hard-failed the launch payload
// on a pristine tree and rewrote prose in committed records (iss-236,
// iss-2609061504302157). The list is built in and not configurable: the
// per-repo pii.json is committed content, and a planted entry there would
// disarm the username gate for a real person's login.
var genericAccountNames = map[string]bool{
	"admin": true, "administrator": true, "app": true, "build": true,
	"builder": true, "ci": true, "codespace": true, "debian": true,
	"deploy": true, "dev": true, "developer": true, "docker": true,
	"ec2-user": true, "git": true, "gitpod": true, "guest": true,
	"jenkins": true, "node": true, "root": true, "runner": true,
	"test": true, "tester": true, "ubuntu": true, "user": true,
	"vagrant": true, "vscode": true, "worker": true,
}

// maxGenericAccountRunes is the length floor under which an account name is too
// short to identify anyone: a one- or two-rune word ("me", "io") collides with
// prose everywhere and names no one.
const maxGenericAccountRunes = 2

// isGenericAccountName reports whether an account name is under the generic
// floor (genericAccountNames, or maxGenericAccountRunes or shorter).
func isGenericAccountName(name string) bool {
	if name == "" {
		return false
	}
	if utf8.RuneCountInString(name) <= maxGenericAccountRunes {
		return true
	}
	return genericAccountNames[strings.ToLower(name)]
}

// accountRootPrefixes are the spellings that put the next segment in the
// account-name position of a home directory: POSIX, Windows, and the
// dash-encoded form a harness uses to name a per-project directory.
var accountRootPrefixes = []string{"/users/", "/home/", `\users\`, "-users-", "-home-"}

// standsAsAccountName reports whether line[start:end] stands where an account
// name stands rather than as a word: the segment after a home root, a tilde
// user ("~name"), or inside the local part of an address or login
// ("name@host", "name.surname@example.com"), or closing the caller's own home
// literal wherever it sits ("…0/root/deck.key" under HOME=/root, which
// home_path_self's leading anchor declines). These are the positions a real
// home path or login leaks from, so a generic account name is still reported
// there, at its hard_fail floor.
//
// Every test reads a bounded window beside the match — the home literal's
// length behind its end, each prefix's length behind its start, at most
// maxLocalPart bytes ahead — and folds only that window. Lower-casing the
// whole line prefix for every match made a line dense in a generic login cost
// the square of its length (iss-2609251535090117).
func standsAsAccountName(line string, start, end int, home string) bool {
	if home != "" && endsWithFold(line[:end], home) {
		return true
	}
	for _, p := range accountRootPrefixes {
		if endsWithFold(line[:start], p) {
			return true
		}
	}
	if start > 0 && line[start-1] == '~' {
		return true
	}
	hi := end
	for hi < len(line) && hi-end < maxLocalPart && isLocalPartByte(line[hi]) {
		hi++
	}
	scanMeter.charge(stageIdentity, hi-end)
	return hi+1 < len(line) && line[hi] == '@' && isAlnumByte(line[hi+1])
}

// maxLocalPart is the longest local part an address can carry (RFC 5321
// section 4.5.3.1.1), and so the furthest standsAsAccountName looks ahead for
// the '@' that makes a match a login.
const maxLocalPart = 64

// endsWithFold reports whether s ends with suffix under Unicode case folding,
// reading only the len(suffix) bytes at the end of s.
func endsWithFold(s, suffix string) bool {
	scanMeter.charge(stageIdentity, len(suffix))
	return len(s) >= len(suffix) && strings.EqualFold(s[len(s)-len(suffix):], suffix)
}

// isLocalPartByte is the byte class of an address's local part as it appears
// in prose: letters, digits and the separators people put in one.
func isLocalPartByte(b byte) bool {
	return isAlnumByte(b) || b == '.' || b == '_' || b == '-' || b == '+' || b == '%'
}

// isNonUserHomeMatch reports whether a generic-home match's final segment is a
// well-known non-user directory under a /Users root.
func isNonUserHomeMatch(matched string) bool {
	scanMeter.charge(stageIdentity, len(matched))
	if !strings.HasPrefix(strings.ToLower(matched), "/users/") {
		return false
	}
	i := strings.LastIndexByte(matched, '/')
	return i >= 0 && IsNonUserHomeSegment(matched[i+1:])
}

// nextPathSegmentEnd returns the end offset of a NAME-BEARING path segment that
// follows pos AND was reached through a traversal segment — a dots-only or empty
// one ("/Users/Shared/../<user>", "/Users/Shared//<user>"). Those walk back out
// of the system root, so the name after them is in the username position again
// and must not be shielded.
//
// It reports false for everything else. "/Users/Shared" and "/Users/Shared/" have
// no following segment; "/Users/Shared/..." is prose with an ellipsis; and
// "/Users/Shared/<seg>/x" reached directly is an entry inside a shared folder
// rather than a home directory, which is the subtree the exemption now covers
// (iss-2609100505145554).
//
// genericHomeRe is POSIX-only, so '/' is the only separator that can reach here.
func nextPathSegmentEnd(line string, pos int) (int, bool) {
	traversed := false
	from := pos
	for pos < len(line) && line[pos] == '/' {
		i, named := pos+1, false
		for i < len(line) && isHomeSegmentByte(line[i]) {
			if line[i] != '.' {
				named = true
			}
			i++
		}
		if named {
			scanMeter.charge(stageIdentity, i-from)
			return i, traversed
		}
		// The segment names nothing: pure dots, or empty (two separators in a
		// row). Either is the escape out of the system root.
		traversed = true
		pos = i
	}
	scanMeter.charge(stageIdentity, pos-from)
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
	scanMeter.charge(stageIdentity, len(matched))
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
func (m identityMatchers) localSuppressionSpans(line string, urls urlSet) []span {
	spans := urls.spans()
	if m.homeSelf != nil {
		scanMeter.charge(stageIdentity, len(line))
		for _, loc := range m.homeSelf.FindAllStringIndex(line, -1) {
			if !homeSweepable(line, loc[0], loc[1], urls) {
				continue // not reported as the home, so it must not suppress the username either
			}
			spans = append(spans, span{loc[0], loc[1]})
		}
	}
	if m.email != nil {
		scanMeter.charge(stageIdentity, len(line))
		for _, loc := range m.email.FindAllStringIndex(line, -1) {
			spans = append(spans, span{loc[0], loc[1]})
		}
	}
	return mergeSpans(spans)
}

// encodedMatches finds the path-encoded username with the ported custom
// boundary: preceded by start-of-string or a non-[A-Za-z0-9.] rune (the RE2
// lookbehind replacement) and followed by EOL or a non-[A-Za-z0-9.] rune.
func encodedMatches(line, encoded string) [][]int {
	var out [][]int
	scanMeter.charge(stageIdentity, len(line))
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

// isDottedNamespaceComponent reports whether line[start:end] is one WHOLE
// component of a dotted, reverse-DNS-shaped identifier — a bundle id, a Java or
// Swift package, a Go module path's host, a domain name — rather than a mention
// of the caller's account.
//
// It is the second structural suppression on the bare-username matcher, the
// sibling of isSystemPathSegment above, and it exists for the same reason: a
// login is a very short word, and a very short word collides. The leading
// component of a reverse-DNS identifier is drawn from a handful of them — com,
// io, app, net, org, me, sh, dev — and every one is a plausible Unix login, so a
// maintainer whose account name is one of them could not write their own bundle
// identifier into a capture without the redactor rewriting it
// (iss-2609100505142469).
//
// The suppression is made rather than merely reported because the damage is
// UNRECOVERABLE: the placeholder does not say which word it replaced, the
// capture is often the only place the identifier was written down, and the
// written record looks clean. Everywhere else abcd fails closed and says so; here
// it corrupted and said nothing. So the collision is made impossible instead.
//
// Three conditions keep it from disarming a genuine leak. The match must be an
// ENTIRE component ("dev" inside "my-dev-tool.a.b" is still flagged); the run
// must have at least THREE components, so a filename ("dev.log") and a
// two-label host are untouched; and a run followed by '@' is an address's local
// part, where the mailbox is the identity, so that stays a leak too. A bare word
// in prose has no dots at all and is unaffected — the ordinary-dictionary-word
// over-redaction (iss-2609061504302157) is a different finding, answered by
// the generic-account floor (isGenericAccountName).
func isDottedNamespaceComponent(line string, start, end int) bool {
	lo, hi := start, end
	for lo > 0 && start-lo < maxDottedIdentifier && isDottedIdentifierByte(line[lo-1]) {
		lo--
	}
	for hi < len(line) && hi-end < maxDottedIdentifier && isDottedIdentifierByte(line[hi]) {
		hi++
	}
	scanMeter.charge(stageIdentity, 2*(hi-lo))
	// A run longer than any identifier is not one, and walking it whole for
	// every login inside it cost the run's length per match
	// (iss-2609251535277823): the bound leaves the finding standing.
	if (lo > 0 && isDottedIdentifierByte(line[lo-1])) || (hi < len(line) && isDottedIdentifierByte(line[hi])) {
		return false
	}
	// A dotted local part is an address, not a namespace.
	if hi < len(line) && line[hi] == '@' {
		return false
	}
	whole, components := false, 0
	for i := lo; i <= hi; {
		j := i
		for j < hi && line[j] != '.' {
			j++
		}
		if j == i {
			// An empty component. A leading or trailing one is the punctuation a
			// sentence leaves behind ("…com.acme.app." at a full stop); an interior
			// one ("com..acme") is not an identifier at all.
			if i != lo && i != hi {
				return false
			}
		} else {
			components++
			if i == start && j == end {
				whole = true
			}
		}
		i = j + 1
	}
	return whole && components >= 3
}

// maxDottedIdentifier is the longest stretch either side of a match that
// isDottedNamespaceComponent reads: a domain name is at most 253 bytes, and a
// bundle id or package path is far shorter.
const maxDottedIdentifier = 255

// isDottedIdentifierByte reports whether b can be part of a dotted identifier —
// the component bytes plus the '.' that separates them. It is deliberately
// narrower than isPathSegmentByte (no '/'): a path is judged by the path
// suppressions, and a namespace by this one.
func isDottedIdentifierByte(b byte) bool {
	return b == '.' || b == '-' || b == '_' ||
		(b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
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

// underAbsoluteRoot reports whether a /Users or /home segment at byte offset
// start, which continues a longer path, sits in an ABSOLUTE local path: the
// path token it belongs to begins with '/' — a backup volume or a mount
// ("/Volumes/Backup/Users/<name>", "/mnt/data/home/<name>") — or is a file URL
// with an empty authority ("file:///home/<name>"). Such a segment is a home
// directory wherever it falls, and leadingBoundaryOK alone let every one of
// them through every detector (iss-324, iss-2608291915432717). A RELATIVE
// token ("docs/Users/guide.md") is not a home, and neither is the path of a
// web URL ("https://docs.example.com/home/…"), whose authority is a host
// rather than this machine: both stay declined, which is the false-positive
// surface the '/'-bearing isPathSegmentByte was guarding.
//
// The token start comes from toks, which walks the line once for every match
// on it: walking back from each match to its token's start cost the token's
// length per match, so a path of nested homes cost the square of its length
// (iss-2609251535090117).
func underAbsoluteRoot(line string, start int, toks *pathTokens) bool {
	i := toks.startOf(start)
	if line[i] != '/' {
		return false
	}
	if i > 0 && line[i-1] == ':' && strings.HasPrefix(line[i:], "//") {
		return strings.HasPrefix(line[i:], "///")
	}
	return true
}

// pathTokens finds the start of the path token holding an offset — the first
// byte of the run of isPathSegmentByte bytes that reaches it — for offsets
// asked in increasing order, walking the line forward once across all of them.
// An offset below the last one asked restarts the walk from the line's start.
type pathTokens struct {
	line       string
	pos, start int
}

// startOf returns the start of the path token that position p continues: p
// itself when the byte before it is not a path byte.
func (t *pathTokens) startOf(p int) int {
	if p < t.pos {
		t.pos, t.start = 0, 0
	}
	scanMeter.charge(stageIdentity, p-t.pos+1)
	for ; t.pos < p; t.pos++ {
		if !isPathSegmentByte(t.line[t.pos]) {
			t.start = t.pos + 1
		}
	}
	return t.start
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
