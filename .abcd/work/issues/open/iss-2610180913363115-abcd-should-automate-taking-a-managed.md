---
schema_version: 1
id: "iss-2610180913363115"
slug: "abcd-should-automate-taking-a-managed"
severity: "minor"
category: "process"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610121549070006 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "proposed: abcd launch public (or abcd ahoy public), building on abcd banlist, abcd lint, abcd launch --dry-run"
remedy: "A staged, resumable go-public flow: prepare a fresh-history public repository from a private sandbox, run the gates, stop for the person to switch visibility, then apply and verify the public-only settings."
---

abcd should automate taking a managed repository public: prepare, gate, hand over the visibility switch, then apply settings

## What happened

On 2026-10-12 a private abcd-managed sandbox was turned into a public product repository (intentdriven/PG-Engine). It took one long session of manual steps, several of them easy to get wrong, and the person stepped in four times: a scope refresh, two banlist decisions, and a force-push. abcd has the pieces (banlist layers, lint, ahoy, launch --dry-run), but nothing chains them.

The steps, in order:

1. **Fresh history.** Create the target repository private, then copy the sandbox's working tree, uncommitted record edits included, and the machine-local tier (the private banlist) as one new root commit. The sandbox stays as it is.
2. **Rename.** The product name changed, so the project name was replaced everywhere: config `project_name`, brief, ADRs, intents, notes and README.
3. **Banlist move.** Moving every third-party pattern from the public layer to the private one meant:
   - converting PCRE to the guard's ERE dialect by hand;
   - verifying each converted pattern against a corpus (91 keys, 12,753 line hits, 0 mismatches);
   - parking the case-sensitive entries the private layer cannot express (rpt-2610121532502681);
   - finding that abcd's own scaffolded hook text ("a link can point") trips a case-insensitive entry.
4. **Truth edits.** The brief's naming rule was rewritten to the new state, in present tense.
5. **Pre-public files.**
   - The licence holder was corrected (it read "abcd").
   - `repo.visibility` was set to public.
   - A CI workflow was added: pinned gitleaks and zizmor, copied from a sibling repository.
   - A CODEOWNERS file was added over the publish surface.
6. **Gates.** `abcd lint`; gitleaks over the tree and the history; the original patterns run over every file and path; a check that the private repository's name appears nowhere.
7. **Commit hygiene.** The first commit's message named the private sandbox, against the PII rule. Fixing it meant amending the root commit and a person-run force-push, because the guard rightly refuses force-pushes from agents. It also meant deleting the first CI run, which would have exposed the old commit's hash and message once public. The orphaned commit stays reachable by hash until GitHub collects it.
8. **ahoy** was run in the new checkout (setup version, gitignore fence, development README).
9. **Visibility switch**, done by the person.
10. **Post-public settings**, which the Free plan blocks on private repositories:
    - secret scanning, push protection and Dependabot;
    - rulesets copied from intentdriven/abcd, with the required checks re-pointed at the jobs this repository actually has;
    - release-tag protection;
    - a check of the effective branch rules and of CODEOWNERS.

## Proposal

A staged, resumable flow. Each stage prints what it will do, applies nothing without a yes, and records its progress in the local tier, so the flow survives the pause while the person switches visibility.

- **prepare:** a fresh-history copy from a source checkout; an optional rename across the record; the banlist migration, converting dialects and proving equivalence per key, and parking or refusing what the private layer cannot express; the minimal CI and CODEOWNERS; the licence holder; the declared visibility.
- **gate:** lint, a secret scan of the tree and the history, the source's original banlist over the new tree, the private source's name absent from files and commit messages, and the Assisted-by trailer present.
- **handover:** stops and asks the person to switch visibility (never done by abcd), naming what will be visible: Actions runs, commit hashes, issues.
- **settle:** applies and verifies the public-only settings (the audit and apply step proposed in rpt-2610121533020069), deletes CI runs that reference rewritten commits, and reports anything left over, such as an orphaned commit only GitHub Support can purge.

## Acceptance

- Run against a private sandbox, the flow produces a public repository whose history, CI runs and files never name the sandbox or any banned term, and the person acts only at the handover and on each yes.
- A second run against the same target finds nothing to do.

Remedy the reporter proposes: A staged, resumable go-public flow: prepare a fresh-history public repository from a private sandbox, run the gates, stop for the person to switch visibility, then apply and verify the public-only settings.

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610121549070006, a enhancement against abcd v0.13.4, surface proposed: abcd launch public (or abcd ahoy public), building on abcd banlist, abcd lint, abcd launch --dry-run.

Evidence:

- rpt-2610121549070006 (the report, kept in the inbox)
- https://github.com/intentdriven/PG-Engine
- rpt-2610121533020069
- rpt-2610121532502681
