package scanner

import (
	"regexp"
	"strings"
)

// Pattern is one compiled secret regex plus its metadata. Skip is the RE2
// lookaround replacement: when non-nil it is called with the full match and,
// when it returns true, the match is discarded (mirrors the negative lookaheads
// the Python patterns used, which RE2 cannot express).
type Pattern struct {
	Name     string
	Kind     string
	Label    string
	Re       *regexp.Regexp
	Severity Severity
	Skip     func(match string) bool // nil == no skip
	// SkipAt is the context-aware form of Skip: it receives the whole line and
	// the match's half-open byte span, so a pattern can reject a match by what
	// SURROUNDS it — the thing RE2 cannot express and Skip cannot see. The
	// network patterns need it to tell a host from a filename (".work.local/")
	// and a quad from a longer dotted run ("1.2.3.4.5.6"). nil == no skip.
	SkipAt     func(line string, start, end int) bool
	Suggestion string
}

// awsExample is the canonical AWS-docs example key, widely used as a test
// placeholder; it must never be flagged (ported neg-lookahead).
const awsExample = "AKIAIOSFODNN7EXAMPLE"

// rpRedactedPlaceholder is the sanitised RepoPrompt sessionKey value; a match
// carrying it is already redacted and must not be re-flagged.
const rpRedactedPlaceholder = "<RP-SESSION-UUID-REDACTED>"

// kindPEMPrivateKey is the one bundled secret kind whose finding is a BLOCK
// rather than a token: Redact consumes the body lines after it (pem.go) and
// masks its span whole (maskedWhole).
const kindPEMPrivateKey = "token:pem_private_key"

// pemPrivateKeyPattern assembles the bundled pem_private_key regex from named
// pieces: the two armour markers, the separator a one-line rendering puts
// between body chunks (a blank, a tab, or the literal \n / \r escape of a JSON
// or YAML dump), a body chunk long enough to be key material, and the short
// final padding chunk a real body ends on.
//
// The three alternatives are ordered so a CLOSED block wins: only there is a
// short final chunk safe unconditionally, because the END marker after it is
// the evidence that it belongs to the key. In an open block the short chunk
// must end the line instead — otherwise the rule is back to claiming prose
// words, which is what ate the sentence around a pasted key.
func pemPrivateKeyPattern() string {
	const (
		begin = `-----BEGIN (?:[A-Z0-9]+ )*PRIVATE KEY(?: BLOCK)?-----`
		end   = `-----END (?:[A-Z0-9]+ )*PRIVATE KEY(?: BLOCK)?-----`
		sep   = `(?:[ \t]|\\[nr])`
		chunk = `[A-Za-z0-9+/=]{16,}`
		short = `[A-Za-z0-9+/=]{1,15}`
	)
	body := sep + `*` + chunk + `(?:` + sep + `+` + chunk + `)*`
	closed := body + `(?:` + sep + `+` + short + `)?` + sep + `*` + end
	open := body + `(?:` + sep + `+` + short + `$)?`
	empty := sep + `*` + end
	return begin + `(?:` + closed + `|` + open + `|` + empty + `)?`
}

// DefaultPatterns returns the bundled secret pattern set (spec §2.2), ported
// verbatim from scripts/abcd/defaults/pii.json, plus the network-identifier set
// from network.go. Every secret pattern is hard_fail and non-sanitisable. This
// set is the built-in baseline the merged config layers on top of (the Go
// analogue of the bundled defaults/pii.json).
func DefaultPatterns() []Pattern {
	p := []Pattern{
		{
			Name: "rp_session_key", Kind: "rp_session_key",
			Label: "RepoPrompt sessionKey UUID",
			// Match the whole key/value; the negative lookahead on the value is
			// ported as a Skip that discards the already-redacted placeholder.
			Re:         regexp.MustCompile(`"sessionKey"\s*:\s*"([^"]+)"`),
			Severity:   SeverityHardFail,
			Skip:       func(m string) bool { return strings.Contains(m, rpRedactedPlaceholder) },
			Suggestion: "RP local-workspace session token — remove or redact to placeholder",
		},
		{
			// No trailing \b (same reasoning as google_api_key below, other axis):
			// the charset is pure word chars but EXCLUDES '_'. Since '_' is itself a
			// word char, a secret immediately followed by '_' (token=ghp_..._old, a
			// JSON key, a concatenation) has no ASCII word boundary after the last
			// alnum, and RE2 cannot shorten to reach one (every interior position is
			// word/word), so a trailing \b silently drops the whole match and the
			// hard_fail secret survives. The leading \b + the ghp_ prefix + the
			// {36,} minimum still bound the match; greedy matching redacts the full
			// token.
			Name: "github_pat", Kind: "token:github_pat", Label: "GitHub PAT (ghp_)",
			Re: regexp.MustCompile(`\bghp_[A-Za-z0-9]{36,}`), Severity: SeverityHardFail,
			Suggestion: "DELETE AND ROTATE — never commit credentials",
		},
		{
			Name: "github_server_token", Kind: "token:github_server", Label: "GitHub server token (ghs_)",
			Re: regexp.MustCompile(`\bghs_[A-Za-z0-9]{36,}`), Severity: SeverityHardFail,
			Suggestion: "DELETE AND ROTATE",
		},
		{
			Name: "github_oauth", Kind: "token:github_oauth", Label: "GitHub OAuth (gho_)",
			Re: regexp.MustCompile(`\bgho_[A-Za-z0-9]{36,}`), Severity: SeverityHardFail,
			Suggestion: "DELETE AND ROTATE",
		},
		{
			Name: "github_user_token", Kind: "token:github_user", Label: "GitHub user token (ghu_)",
			Re: regexp.MustCompile(`\bghu_[A-Za-z0-9]{36,}`), Severity: SeverityHardFail,
			Suggestion: "DELETE AND ROTATE",
		},
		{
			Name: "github_refresh", Kind: "token:github_refresh", Label: "GitHub refresh token (ghr_)",
			Re: regexp.MustCompile(`\bghr_[A-Za-z0-9]{36,}`), Severity: SeverityHardFail,
			Suggestion: "DELETE AND ROTATE",
		},
		{
			// Fine-grained PAT, GitHub's default token type since 2022:
			// github_pat_ + 22 alnum + '_' + 59 alnum. The classic ghp_ pattern
			// above cannot match this prefix, so it needs its own entry.
			Name: "github_pat_finegrained", Kind: "token:github_pat_finegrained",
			Label:      "GitHub fine-grained PAT (github_pat_)",
			Re:         regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9]{22}_[A-Za-z0-9]{59}`),
			Severity:   SeverityHardFail,
			Suggestion: "DELETE AND ROTATE — never commit credentials",
		},
		{
			// PEM private-key block. The scanner is line-oriented and the BEGIN
			// line is single-line and self-identifying, so the header is what
			// flags the block (RSA/EC/DSA/OPENSSH/PGP/ENCRYPTED/plain). The match
			// then reaches over a body that shares the header's LINE — base64
			// chunks separated by blanks or the literal \n and \r escapes of a
			// JSON or YAML dump — through an END marker on that line when there
			// is one, so Redact masks the body bytes a one-line key carries and
			// not just the header before them. EVERY chunk the body claims must
			// be long enough to be key material: a chunk rule that took any
			// word-run once a body had opened ate the sentence a paste was
			// embedded in ("… and it was rotated on Tuesday" went with the key).
			// A SHORT final chunk — a real body's padding line — is taken only
			// where the evidence for it is there: an END marker closes the block,
			// or the chunk ends the line. Body lines of their own belong to the
			// block consumer (pem.go); the span is masked whole (maskedWhole).
			Name: "pem_private_key", Kind: kindPEMPrivateKey,
			Label:      "PEM private key",
			Re:         regexp.MustCompile(pemPrivateKeyPattern()),
			Severity:   SeverityHardFail,
			Suggestion: "DELETE AND ROTATE — private key material must never be committed",
		},
		{
			Name: "anthropic_key", Kind: "token:anthropic", Label: "Anthropic API key (sk-ant-)",
			Re: regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{40,}`), Severity: SeverityHardFail,
			Suggestion: "DELETE AND ROTATE",
		},
		{
			Name: "openai_project_key", Kind: "token:openai_project", Label: "OpenAI project key (sk-proj-)",
			Re: regexp.MustCompile(`\bsk-proj-[A-Za-z0-9_-]{40,}`), Severity: SeverityHardFail,
			Suggestion: "DELETE AND ROTATE",
		},
		{
			Name: "openai_service_account", Kind: "token:openai_svcacct", Label: "OpenAI service account key (sk-svcacct-)",
			Re: regexp.MustCompile(`\bsk-svcacct-[A-Za-z0-9_-]{40,}`), Severity: SeverityHardFail,
			Suggestion: "DELETE AND ROTATE",
		},
		{
			Name: "aws_access_key", Kind: "token:aws_access_key", Label: "AWS access key ID",
			// The prefix set is the documented AWS access-key-ID family, not the
			// single long-term AKIA literal (gh-358): the canonical gitleaks
			// aws-access-token form
			// `(A3T[A-Z0-9]|AKIA|ASIA|ABIA|ACCA)[A-Z0-9]{16}`. ASIA (temporary STS
			// key), ABIA (STS service bearer token), ACCA (context-specific
			// credential) and the legacy A3T access key are all shape-identical
			// credential material that must block exactly as AKIA does. The
			// RESOURCE-identifier prefixes (AROA role, AIDA user, AGPA group, AIPA
			// instance profile, ANPA/ANVA policy, APKA public key, ASCA certificate)
			// are DELIBERATELY excluded: they are non-secret identifiers that appear
			// openly in ARNs and policy documents, so matching them would be a false
			// positive. The neg-lookahead excluding the docs example is ported as a Skip.
			Re:         regexp.MustCompile(`\b(?:A3T[0-9A-Z]|AKIA|ASIA|ABIA|ACCA)[0-9A-Z]{16}`),
			Severity:   SeverityHardFail,
			Skip:       func(m string) bool { return m == awsExample },
			Suggestion: "DELETE AND ROTATE — also rotate corresponding secret in IAM",
		},
		{
			// No trailing \b: the charset [A-Za-z0-9-] excludes '_' (a word char), so
			// a token followed by '_' had the same miss as the ghp_ family above; and
			// the mixed '-' also has the google_api_key axis. No token pattern in this
			// set relies on a trailing \b — the leading \b + prefix + minimum length
			// bound the match, greedy matching redacts the full token.
			Name: "slack_token", Kind: "token:slack", Label: "Slack token (xox*)",
			Re: regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}`), Severity: SeverityHardFail,
			Suggestion: "DELETE AND ROTATE",
		},
		{
			Name: "google_api_key", Kind: "token:google_api", Label: "Google API key (AIza)",
			// No trailing \b: the class includes '-' and the length is FIXED at 35, so
			// a key whose 35th char is '-' (a valid Google key char) has no shorter
			// match to fall back on and the ASCII \b — which needs a word char on its
			// left — can never be satisfied, silently missing a hard_fail secret. The
			// leading \b plus the AIza prefix and fixed length still bound the match.
			Re: regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`), Severity: SeverityHardFail,
			Suggestion: "DELETE AND ROTATE",
		},
		{
			Name: "stripe_live_key", Kind: "token:stripe_live", Label: "Stripe live key (sk_live_)",
			Re: regexp.MustCompile(`\bsk_live_[A-Za-z0-9]{20,}`), Severity: SeverityHardFail,
			Suggestion: "DELETE AND ROTATE — production credential",
		},
		{
			Name: "stripe_test_key", Kind: "token:stripe_test", Label: "Stripe test key (sk_test_)",
			Re: regexp.MustCompile(`\bsk_test_[A-Za-z0-9]{20,}`), Severity: SeverityHardFail,
			Suggestion: "Review — test keys are lower risk but still shouldn't be committed",
		},
		{
			Name: "jwt_shaped", Kind: "token:jwt_shaped", Label: "JWT-shaped token",
			Re:         regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`),
			Severity:   SeverityHardFail,
			Suggestion: "Review — may be benign sample or real bearer token",
		},
	}
	// Network identifiers are part of the baseline, not a bolt-on: folding them
	// in here is what makes every consumer (launch dry-run, lifeboat pack,
	// history Stage-1 redaction) inherit the same detection from one definition.
	// The harness-leak class (harnessleak.go) is folded in on the same reasoning:
	// a session URL and a tool's attribution footer are content that must not
	// leave the machine, and defining them anywhere else would give the
	// store-before-commit paths a weaker notion of a leak than the surfaces that
	// judge committed text.
	p = append(p, NetworkPatterns()...)
	return append(p, HarnessLeakPatterns()...)
}

// defaultPatternFloors captures the built-in severity floor per bundled pattern
// name (used to clamp a config override that tries to downgrade one).
func defaultPatternFloors() map[string]Severity {
	floors := map[string]Severity{}
	for _, p := range DefaultPatterns() {
		floors[p.Name] = p.Severity
	}
	return floors
}
