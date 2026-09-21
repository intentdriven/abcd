---
id: spc-2609211957080006
slug: plugin-metadata-lockstep-update
intent: itd-69
origin: researcher-authored
production_mode: hand-written
---
# plugin-metadata-lockstep-update

## Summary

The design record for itd-69 as delivered in v0.1.0: `launch.CheckLockstep`
reads the version and manifest fields across the plugin's duplicated
metadata surfaces and refuses drift; written on 2026-09-21 to close a record
that shipped without one.

## Scope, as delivered

1. **The check** (`internal/core/launch/lockstep.go`): the ADR-pinned path
   list (`plugin.json` in both locations, `.claude-plugin/marketplace.json`),
   per-field comparison, a distinct result for agreement, drift, non-semver
   and an unreadable contract (criterion 1).
2. **The tree rule**: `TreeDev` requires absent version keys per adr-19 and
   reports a present one as drift; `TreePublic` requires agreement
   (criterion 2).
3. **No bypass**: the check exposes no skip or dirty flag; `launch --dry-run`
   runs it unconditionally and the release gate reads its result
   (criterion 3).

## How the criteria are satisfied

All three by scope 1 to 3 as shipped (`lockstep_test.go`: agreement, drift,
non-semver, contract unreadable, dev keys absent).
