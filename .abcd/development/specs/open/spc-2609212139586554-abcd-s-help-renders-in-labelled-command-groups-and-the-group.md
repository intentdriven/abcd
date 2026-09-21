---
id: spc-2609212139586554
slug: abcd-s-help-renders-in-labelled-command-groups-and-the-group
intent: itd-146
origin: researcher-authored
production_mode: hand-written
---
# abcd-s-help-renders-in-labelled-command-groups-and-the-group

## Summary

The design record for itd-146 as widened on 2026-09-21: grouped help for people, an agents-and-hosts block behind `--agent`, the snapshot recording both.

## Scope

1. **Groups and blocks** on the cobra root: `AddGroup` per group, a `block` annotation per command (`people` default, `agents` where decision 2 says), the help template rendering the person's groups and the expanding line by default and both blocks with `--agent` (criteria 1, 2).
2. **Execution unchanged**: no `Hidden`; a test registers a verb with no group and asserts failure (criterion 3).
3. **The snapshot**: `group` and `block` fields, schema version bumped, the regeneration gate (criterion 4).
4. **The pages**: each `commands/*.md` frontmatter gains `block:`; the agent block's help lines print the page path (criterion 5).
5. **Places kept**: `rules`, `spec` untouched (criterion 6).

## Out of scope

- Renames and merges (itd-2609212130136102); the sentence per verb (itd-2609212113220149).

## Approach

Cobra's group support plus one annotation and a custom help template; the surface manifest generator already walks the command tree and gains two fields.

## Footprint

- packages: internal/surface/cli, internal/core/surface
- tests: the rendered help with and without --agent; the no-group failure; the snapshot diff gate; the page frontmatter check

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 grouped list and the line | scope 1 |
| 2 two blocks | scope 1 |
| 3 runs the same; no-group fails | scope 2 |
| 4 snapshot | scope 3 |
| 5 pages | scope 4 |
| 6 places kept | scope 5 |
