package rules

import "github.com/intentdriven/abcd/internal/core/guard"

// ShellDomain is the bundled domain that carries itd-103's teaching plane
// (spc-16, "Two planes, one registry"; iss-151, ruling J10): the shell-hazard
// registry's lessons, injected before shell-heavy work so an agent is taught
// the safe form up front. A host without hook support gets this plane alone; a
// host with hooks gets it and the guard.
//
// The domain is GENERATED from the bundled registry the guard reads, never
// written in defaults/rules.json: one rule per registry entry (the entry's
// Lesson, in id order) and one recall term per command head the registry
// matches. An entry added to or removed from the registry changes the domain
// with no second edit, and TestShellDomainIsGeneratedFromTheGuardRegistry fails
// if the two ever part. It is an ordinary bundled domain to every loader
// contract — per-field user and repo overrides, dormant, the kill switch, the
// *SHELL star command, dedup and provenance — and a guardrail domain to
// noteWithheld.
//
// It is built from the BUNDLED registry only. A repo's .abcd/guard.json
// changes what the guard refuses in that repo, and the two features keep
// independent switches (spc-16, "Config home"); a repo that wants its own
// entries taught says so in its rules.json, as for any bundled domain.
const ShellDomain = "SHELL"

// shellAliases is the fixed half of the domain's recall: words that name
// shell-heavy work without naming a command the registry matches. The
// registry offers the command heads (guard.Registry.RecallTerms); it has no
// word for "a shell session" or for a force push asked for in prose, so these
// few are declared here. Kept narrow on purpose: "push", "commit", "reset" and
// "terminal" are ordinary vocabulary in a repository's prose and never recall
// the domain alone.
var shellAliases = []string{"bash", "command line", "force push", "shell", "zsh"}

// shellDomain generates the teaching-plane domain from reg. ok is false when
// the registry has no entries: a domain with no rules would render as a heading
// with nothing under it, the shape Validate refuses.
func shellDomain(reg guard.Registry) (Domain, bool) {
	lessons := reg.Lessons()
	if len(lessons) == 0 {
		return Domain{}, false
	}
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
	if d, ok := shellDomain(guard.Defaults()); ok {
		if rs.Domains == nil {
			rs.Domains = map[string]Domain{}
		}
		rs.Domains[ShellDomain] = d
	}
	return rs
}
