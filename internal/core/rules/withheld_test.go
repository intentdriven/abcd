//go:build unix

package rules

import (
	"strings"
	"testing"
)

// networkRule is the PII rule iss-156 added to the bundled defaults — the
// upgrade a repo that pinned PII's rules before it never received.
const networkRule = "Never commit hostnames, IP addresses, MAC addresses, or other live network identifiers"

// withheldNote returns the one note naming domain as withholding field, or "".
func withheldNote(rs RuleSet, domain, field string) string {
	for _, n := range rs.Notes() {
		if strings.Contains(n, `"`+domain+`"`) && strings.Contains(n, "WITHHOLDS") && strings.Contains(n, field) {
			return n
		}
	}
	return ""
}

// TestLoadNamesTheBundledSecurityRulesAnOverrideWithholds (iss-174): a repo
// that pinned PII's rules before the network-identifier rule shipped keeps the
// old list — per-field replacement is the documented merge — but it may not do
// so SILENTLY. The load names the bundled rule the override is withholding, and
// the file that withholds it, so a stale security rule set never looks current.
func TestLoadNamesTheBundledSecurityRulesAnOverrideWithholds(t *testing.T) {
	repo := t.TempDir()
	writeRepoRules(t, repo, `{"schema_version":1,"domains":{"PII":{"rules":[
		"Never commit, print, or paste secrets, tokens, or .env contents; reference them by name.",
		"Never put absolute local paths in anything that leaves the machine — use repo-relative paths.",
		"Never name a private repo in commits, PRs, issues, or docs; describe it generically."]}}}`)

	rs, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := len(rs.Domains["PII"].Rules); got != 3 {
		t.Fatalf("the merge semantics moved: PII carries %d rules, want the override's 3", got)
	}
	note := withheldNote(rs, "PII", "rules")
	if note == "" {
		t.Fatalf("the withheld bundled rule is silent; notes = %q", rs.Notes())
	}
	for _, want := range []string{RepoRelPath, networkRule} {
		if !strings.Contains(note, want) {
			t.Errorf("the note does not name %q: %s", want, note)
		}
	}
	if strings.Contains(note, "Never name a private repo") {
		t.Errorf("the note names a bundled rule the override restates: %s", note)
	}
}

// TestLoadNamesTheBundledRecallAnOverrideWithholds: withholding recall is the
// quieter half — the rules are all there, but prompts that name the network no
// longer summon them. The keywords are named like the rules are.
func TestLoadNamesTheBundledRecallAnOverrideWithholds(t *testing.T) {
	repo := t.TempDir()
	writeRepoRules(t, repo, `{"schema_version":1,"domains":{"PII":{"recall":["secret","token","credential","pii","redact"]}}}`)

	rs, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	note := withheldNote(rs, "PII", "recall")
	if note == "" {
		t.Fatalf("the withheld bundled recall keywords are silent; notes = %q", rs.Notes())
	}
	for _, want := range []string{`"tailscale"`, `"hostname"`} {
		if !strings.Contains(note, want) {
			t.Errorf("the note does not name the withheld keyword %s: %s", want, note)
		}
	}
	if strings.Contains(note, `"secret"`) {
		t.Errorf("the note names a keyword the override keeps: %s", note)
	}
}

// TestLoadNamesTheUserLayerThatWithholds: the user scope's file replaces a
// field the same way, and the note must name the file that did it — the one to
// edit — rather than the repo's.
func TestLoadNamesTheUserLayerThatWithholds(t *testing.T) {
	home := userHome(t)
	writeUserRules(t, home, `{"schema_version":1,"domains":{"LOAD":{"rules":["Ask before a load experiment."]}}}`)

	rs, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	note := withheldNote(rs, "LOAD", "rules")
	if note == "" {
		t.Fatalf("the user layer's withholding is silent; notes = %q", rs.Notes())
	}
	if !strings.Contains(note, UserDisplayPath) {
		t.Errorf("the note does not name the user-scope file: %s", note)
	}
}

// TestLoadIsQuietWhenNothingIsWithheld: the note is about loss, not about
// overriding. An override that restates every bundled entry and adds its own,
// one that only changes state, and one that replaces a domain carrying no
// security rule all load without a word — a note on every deliberate override
// would teach the reader to skip the one that matters.
func TestLoadIsQuietWhenNothingIsWithheld(t *testing.T) {
	bundled := Defaults().Domains["PII"]
	superset := `{"schema_version":1,"domains":{"PII":{"rules":[` + quoteAll(append(append([]string(nil), bundled.Rules...), "Mind the widget.")) +
		`],"recall":[` + quoteAll(append(append([]string(nil), bundled.Recall...), "widget")) + `]}}}`
	for name, body := range map[string]string{
		"superset":        superset,
		"state only":      `{"schema_version":1,"domains":{"PII":{"state":"dormant"}}}`,
		"not a guardrail": `{"schema_version":1,"domains":{"ROADMAP":{"rules":["Our own roadmap rule."]}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			repo := t.TempDir()
			writeRepoRules(t, repo, body)
			rs, err := Load(repo)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			for _, n := range rs.Notes() {
				if strings.Contains(n, "WITHHOLDS") {
					t.Errorf("unexpected note: %s", n)
				}
			}
		})
	}
}

// TestSecurityBearingDomainsAreBundled: the list of guarded domains names
// bundled domains only, so a rename in the defaults cannot quietly drop one
// from the check.
func TestSecurityBearingDomainsAreBundled(t *testing.T) {
	d := Defaults()
	for _, name := range securityBearingDomains {
		if _, ok := d.Domains[name]; !ok {
			t.Errorf("securityBearingDomains names %q, which the bundled defaults do not carry", name)
		}
	}
}

func quoteAll(ss []string) string {
	q := make([]string, len(ss))
	for i, s := range ss {
		q[i] = `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return strings.Join(q, ",")
}
