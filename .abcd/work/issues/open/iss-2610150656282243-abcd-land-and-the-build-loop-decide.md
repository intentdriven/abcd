---
schema_version: 1
id: "iss-2610150656282243"
slug: "abcd-land-and-the-build-loop-decide"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "abcd land security review 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/ruleset.go"
remedy: "Before arming a merge, confirm the merge rule read from the committed ruleset mirror against the live forge's branch rules (gh api repos/{owner}/{repo}/rules/branches/{default}); refuse to arm when they disagree, naming both, in abcd land and the build loop alike"
---

abcd land and the build loop decide whether a merge is held for a person's approval (ruling AM1) from the committed ruleset mirror under .abcd/work/rulesets/, never from the forge's live rules. A mirror that has gone stale, saying an approval is required when the forge no longer requires one, lets abcd arm a merge that no person gates. Found by the Opus security reviewer of the abcd land change on 2026-10-15; the loop has the same reading, so this predates abcd land. Route exists without abcd: no, the mirror-based decision is abcd's, so abcd fixes it.

2026-10-16, evidence from a real run in a private throwaway lab with no ruleset mirror: abcd land (v0.15.0) reported "no merge queue gates main (per the ruleset mirror), so its merge is left to a person". With no mirror at all, that claim rests on absence: a repository whose forge does run a merge queue but carries no mirror reads the same way, and the note credits a mirror that does not exist. Reading the live forge's rules, as this record proposes, settles both; until then the note should say that no ruleset mirror exists.

2026-10-30, after the merge gate replaced ruling AM1 (itd-2610150602547271): the mirror no longer decides whether a merge waits for a person, only whether a merge queue gates the default branch. With no mirror, abcd land and the build loop arm `--auto --merge`. The live forge accepts that arming against this repository's merge queue: it answers that the queue sets the merge strategy and enqueues the pull request (seen on two pull requests that day). What a missing mirror loses is land's queue path. Land reports that the forge merges once the required checks pass, and skips the confirmation and the auto-merge toggle that recover a pull request left outside the queue (iss-2610090642376032). A queue whose method is not a merge commit was not tried. The remedy above still settles both.
