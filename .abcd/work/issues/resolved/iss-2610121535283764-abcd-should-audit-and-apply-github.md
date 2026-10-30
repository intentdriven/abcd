---
schema_version: 1
id: "iss-2610121535283764"
slug: "abcd-should-audit-and-apply-github"
severity: "major"
category: "security"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610121533020069 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd ahoy (proposed: abcd ahoy github / --apply)"
remedy: "Extend abcd ahoy --remote / remote apply (today it covers only secret scanning and push protection) into a read-only audit of a managed repository's GitHub settings against a declared baseline (security features, repository hygiene, default-branch and tag rulesets, CODEOWNERS, Actions policy, declared vs real visibility and plan), with an opt-in, idempotent apply that never changes visibility, requires only checks the repository's workflows actually run, records plan-unavailable settings as owed, takes per-repository overrides from .abcd config and mirrors what it applied; plan it as an intent under the security standard rule (never change how a repository works without a yes)"
resolution: "abcd ahoy --remote audits the GitHub settings baseline read-only, and abcd ahoy remote apply applies the differences on a yes and mirrors them"
impact: additive
resolved_by:
  intent: "itd-2610290746077716"
  spec: "spc-2610290749113317"
  commit: "2bb0530796bef1833b1064b9af2e04477fb6249c"
---

abcd should audit and apply GitHub security settings for every repository it manages

## Problem

Settings differ between abcd-managed repositories, and nothing in abcd notices.

- **intentdriven/abcd** has secret scanning, push protection and Dependabot security updates on. It has rulesets for main, release tags and reservations, a merge queue, and a CODEOWNERS file.
- **intentdriven/Dessau** is public, but has secret scanning, push protection and Dependabot security updates off. It has one ruleset ("protect"), no CODEOWNERS, the wiki on, and branches kept after merge.
- **The [redacted-user] org** switches every security feature off for new repositories by default. Actions may use any action and need no SHA pinning; the default workflow token is read-only and cannot approve pull requests. Org-wide rulesets need GitHub Team ("Upgrade to GitHub Team to enable this feature").
- **intentdriven/PG-Engine**, created today as private, could not turn on secret scanning ("Secret scanning is not available for this repository"). Only Dependabot alerts and security updates applied.

## Proposal

A read-only audit, plus an opt-in apply step, that:

1. Reads the declared visibility (`.abcd/config.json` `repo.visibility`) and the real visibility and plan on GitHub, and reports any mismatch. It never changes visibility itself.
2. Compares the repository with a baseline and prints each difference as setting / current / target:
   - security: secret scanning, push protection, Dependabot alerts and security updates;
   - repository: branches deleted after merge, wiki and projects off unless used;
   - default-branch ruleset: no deletion, no force-push, required status checks, a pull-request rule with code-owner review; release tags protected;
   - a CODEOWNERS file covering the publish surface (`.github/`, `.githooks/`, `.abcd/` config, build files);
   - Actions: allow-list or SHA pinning, recommended where the org allows any action.
3. Requires only the checks the repository actually runs, taken from its workflow job names, so it never blocks every merge on a check that does not exist. On a repository with no CI, it proposes a minimal gitleaks plus `abcd lint` workflow and requires those.
4. Skips whatever the plan does not offer and says so (on Free, secret scanning, push protection and rulesets are unavailable for private repositories), recording those settings as owed so they apply once the repository goes public.
5. Takes per-repository overrides from `.abcd/` config, version-controlled and reviewed like other config, and mirrors the applied rulesets into the record so drift can be checked.
6. Is idempotent, applies nothing without a yes, and names any token scope it lacks (admin:org for the org's Actions policy) instead of failing silently.

## Acceptance

- Run against Dessau, the audit lists the differences above; after apply, a second audit is clean.
- Run against a private Free-plan repository, it applies what is available and lists the rest as owed until the repository is public.

Remedy the reporter proposes: Add a read-only audit of a managed repository's GitHub settings against a baseline taken from intentdriven/abcd, with an opt-in apply step that is idempotent, honours the plan's limits and never changes visibility.

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610121533020069, a enhancement against abcd v0.13.4, surface abcd ahoy (proposed: abcd ahoy github / --apply).

Evidence:

- rpt-2610121533020069 (the report, kept in the inbox)
- https://github.com/intentdriven/abcd
- https://github.com/intentdriven/Dessau
- https://github.com/intentdriven/PG-Engine
