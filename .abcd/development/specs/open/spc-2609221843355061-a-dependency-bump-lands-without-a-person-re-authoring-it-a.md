---
id: spc-2609221843355061
slug: a-dependency-bump-lands-without-a-person-re-authoring-it-a
intent: itd-2609221842494980
origin: researcher-authored
production_mode: hand-written
---
# a-dependency-bump-lands-without-a-person-re-authoring-it-a

## Summary

The design record for itd-2609221842494980: the bounded re-authoring workflow, its declaration, its scaffold and its record.

## Scope

1. **The test** (`internal/core/launch` beside the release scaffold, rendered into the workflow): branch matches a declared bot's prefix AND `git diff --name-only` against the base is a subset of that ecosystem's manifest-and-lock pair; anything else fails the test and the run prints which clause failed (criteria 1, 2, 5).
2. **The ecosystems**: a table in the scaffolded configuration, one row per ecosystem (manifest, lock, bot), extended by the repository; an unknown ecosystem matches nothing (criterion 5).
3. **The re-authoring**: `git commit --amend --reset-author` on the bot's head with a message the workflow composes (the original subject, a line naming the bot and the workflow, `Assisted-by: None`), pushed to the same branch under the repository's own credentials (criteria 1, 3).
4. **The gate**: untouched; a test asserts the re-authored commit passes `scripts/check-attribution.sh commits` and the bot's original still fails it (criterion 4).
5. **The scaffold**: `launch scaffold` gains the workflow behind the repository's opt-in, beside the release workflows (criterion 6).
6. **The record**: one line per re-authoring into the repository's own log surface, naming the bot, the bump and the commit (criterion 7).
7. **Review**: security reviewer on the lane, for the credential use and the push (criteria 1, 3).

## Out of scope

- Editing the gate; merging; choosing dependencies.

## Approach

The workflow is composed by the same renderer that writes the release workflows, so a managed repository gets it through the path it already trusts; the bound is one shell test over the diff, which is what makes the rule auditable without trusting the bot.

## Footprint

- packages: internal/core/launch, .github/workflows, scripts
- tests: the bound over fixtures (in-bound bump, out-of-bound bump, unlisted bot, unknown ecosystem); the re-authored commit against the attribution script; the scaffold's opt-in; the record line

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 in-bound bump re-authored | scope 1, 3 |
| 2 anything else untouched, reason named | scope 1, 2 |
| 3 message and trailer | scope 3 |
| 4 gate unchanged and still refuses the bot | scope 4 |
| 5 undeclared bot or ecosystem | scope 2 |
| 6 scaffolded, opt-in | scope 5 |
| 7 the record | scope 6 |
