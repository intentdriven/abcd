package rules

import (
	"fmt"
	"slices"

	"github.com/intentdriven/abcd/internal/core/guard"
)

// ShellDomain is the bundled domain generated from the shell-hazard registry
// the guard reads (itd-103, as narrowed by itd-2611041636196213). It has two
// lists. Rules is what is sent as shell work starts: one line saying a guard
// judges each command, then the lessons of the commands it only warns about,
// since a warned command still runs. Checked is what the guard carries: the
// lessons of what it refuses, which the agent reads at the refusal, in the
// main session and in a sub-agent alike, and `abcd rules SHELL` lists. With the
// guard switched off nothing refuses, so every lesson stays in Rules and no
// line claims a guard.
//
// The domain is GENERATED, never written in defaults/rules.json: an entry added
// to or removed from the registry changes the domain with no second edit, and
// TestEveryLessonIsInExactlyOneList fails if a hazard falls out of both lists.
// The script reading, a variable-built `-c` script that deletes and a delete
// the guard cannot prove safe have no entry of their own, so their lessons are
// the guard's (guard.Registry.ScriptLesson, ShellCLesson, DeleteLesson). One
// recall term stands for each command head the registry matches. To every
// loader contract (per-field overrides, dormant, the kill switch, the *SHELL
// star command, dedup, provenance) it is an ordinary bundled domain, and a
// guardrail domain to noteWithheld.
//
// The bundled defaults carry the domain generated from the bundled registry.
// Load regenerates it, by the same generator, from the registry the guard
// enforces in the repository, the bundled entries merged with the repository's
// own .abcd/guard.json (ruling CK1), each such lesson marked guard.RepoMark. The
// switches stay independent (spc-16, "Config home"): the guard file decides what
// is refused, and rules.json overrides, silences or kills what is sent.
const ShellDomain = "SHELL"

// shellAliases is the fixed half of the domain's recall: words that name
// shell-heavy work without naming a command the registry matches. The
// registry offers the command heads (guard.Registry.RecallTerms); it has no
// word for "a shell session" or for a force push asked for in prose, so these
// few are declared here. Kept narrow on purpose: "push", "commit", "reset" and
// "terminal" are ordinary vocabulary in a repository's prose and never recall
// the domain alone.
var shellAliases = []string{"bash", "command line", "force push", "shell", "zsh"}

// shellDomain generates the domain from reg, marking every lesson bundled does
// not hold word for word as the repository's. ok is false when the registry has
// no entries: a domain with no rules would render as a heading with nothing
// under it, the shape Validate refuses.
func shellDomain(reg, bundled guard.Registry) (Domain, bool) {
	if len(reg.Entries) == 0 {
		return Domain{}, false
	}
	warned, refused := reg.SplitLessons(bundled)
	own := []string{reg.ScriptLesson(), reg.ShellCLesson(), reg.DeleteLesson()}
	d := Domain{
		State:   StateActive,
		Recall:  reg.RecallTerms(),
		Aliases: append([]string(nil), shellAliases...),
	}
	if reg.Disabled {
		d.Rules = append(warned, own...)
		return d, true
	}
	d.Rules = append([]string{guard.SteeringLine}, warned...)
	d.Checked = append(refused, own...)
	return d, true
}

// withShellDomain adds the generated domain to the parsed
// bundled defaults. A rules.json that declares the domain by hand is a build
// error: it would be a second copy of the registry's words, free to drift from
// the first.
func withShellDomain(rs RuleSet) RuleSet {
	if _, ok := rs.Domains[ShellDomain]; ok {
		panic("rules: bundled defaults declare " + ShellDomain + " by hand; it is generated from the guard registry")
	}
	if d, ok := shellDomain(guard.Defaults(), guard.Defaults()); ok {
		if rs.Domains == nil {
			rs.Domains = map[string]Domain{}
		}
		rs.Domains[ShellDomain] = d
	}
	return rs
}

// withRepoShellDomain regenerates the SHELL domain of the bundled set rs from
// the registry the guard enforces in repoRoot (ruling CK1): the bundled
// entries merged with the repository's .abcd/guard.json. It runs before any
// rules.json layer, so the regenerated domain is the base those layers
// override per field, exactly as they override the bundled one.
//
// A guard.json the guard refuses — unreadable, invalid, or an uncommitted
// edit that weakens it — is refused here too, loudly: the domain teaches the
// registry the guard falls back to (the bundled one, or HEAD's committed
// file), never the refused entries, and a note names the file and the reason
// on every load. Failing the whole rule set instead would take PII, COMMITTING
// and every other domain down with one broken guard file; skipping it silently
// would leave a repository believing its own hazard is taught.
func withRepoShellDomain(rs RuleSet, repoRoot string) RuleSet {
	ld := guard.LoadRepo(repoRoot)
	if ld.Err != nil {
		rs.notes = append(rs.notes, fmt.Sprintf(
			"rules: %s: the repository's own guard entries are refused and not taught (%v); %s teaches the registry the guard enforces in their place",
			ShellDomain, ld.Err, ShellDomain))
	}
	if ld.Posture == guard.LoadUnavailable {
		return rs
	}
	if d, ok := shellDomain(ld.Registry, guard.Defaults()); ok {
		if !slices.Equal(d.Rules, rs.Domains[ShellDomain].Rules) {
			rs.setRulesFrom(ShellDomain, guard.RepoRelPath)
		}
		rs.Domains[ShellDomain] = d
	}
	return rs
}
