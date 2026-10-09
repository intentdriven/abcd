package rules

import (
	"fmt"
	"slices"

	"github.com/intentdriven/abcd/internal/core/guard"
)

// ShellDomain is the bundled domain that carries itd-103's teaching plane
// (spc-16, "Two planes, one registry"; iss-151, ruling J10): the shell-hazard
// registry's lessons, injected before shell-heavy work so an agent is taught
// the safe form up front. A host without hook support gets this plane alone; a
// host with hooks gets it and the guard.
//
// The domain is GENERATED from the bundled registry the guard reads, never
// written in defaults/rules.json: one rule per registry entry (the entry's
// Lesson, in id order), then the guard's rule for the scripts it reads
// (guard.Registry.ScriptLesson), and one recall term per command head the
// registry matches. An entry added to or removed from the registry changes the domain
// with no second edit, and TestShellDomainIsGeneratedFromTheGuardRegistry fails
// if the two ever part. It is an ordinary bundled domain to every loader
// contract — per-field user and repo overrides, dormant, the kill switch, the
// *SHELL star command, dedup and provenance — and a guardrail domain to
// noteWithheld.
//
// The bundled defaults carry the domain generated from the bundled registry.
// Load regenerates it, by the same generator, from the registry the guard
// enforces in the repository — the bundled entries merged with the
// repository's own .abcd/guard.json (ruling CK1) — so a repository's own
// hazards are taught as the bundled ones are, each such lesson marked
// guard.RepoMark. The switches stay independent (spc-16, "Config home"): the
// guard file decides what is refused, and rules.json overrides, silences or
// kills the teaching of it like any bundled domain's.
const ShellDomain = "SHELL"

// shellAliases is the fixed half of the domain's recall: words that name
// shell-heavy work without naming a command the registry matches. The
// registry offers the command heads (guard.Registry.RecallTerms); it has no
// word for "a shell session" or for a force push asked for in prose, so these
// few are declared here. Kept narrow on purpose: "push", "commit", "reset" and
// "terminal" are ordinary vocabulary in a repository's prose and never recall
// the domain alone.
var shellAliases = []string{"bash", "command line", "force push", "shell", "zsh"}

// shellDomain generates the teaching-plane domain from reg, marking every
// lesson bundled does not teach word for word as the repository's. ok is false
// when the registry has no entries: a domain with no rules would render as a
// heading with nothing under it, the shape Validate refuses.
func shellDomain(reg, bundled guard.Registry) (Domain, bool) {
	lessons := reg.LessonsOver(bundled)
	if len(lessons) == 0 {
		return Domain{}, false
	}
	// The script reading has no registry entry of its own, so its rule is
	// the guard's, generated beside the reading (guard.ScriptLesson).
	lessons = append(lessons, reg.ScriptLesson())
	return Domain{
		State:   StateActive,
		Recall:  reg.RecallTerms(),
		Aliases: append([]string(nil), shellAliases...),
		Rules:   lessons,
	}, true
}

// withShellDomain adds the generated teaching-plane domain to the parsed
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
