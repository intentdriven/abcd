---
schema_version: 1
id: "iss-2609282105242542"
slug: "the-guard-s-bundled-registry-internal"
severity: "major"
category: "security"
source: "review-followup"
found_during: "v0.11.1 release gate crosscheck (autonomous run A)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
deferred_after: v0.11.0
deferral_reason: "Found by the v0.11.1 release gate after the content commit; the gap already shipped in v0.11.0 (registry unchanged), and a guard-registry change needs its own tests and a security review, so it is fixed in the first lane after the v0.11.1 tag rather than re-rolling this cut"
resolution: "Two bundled guard entries through a new arg_values pattern field: rm-rf-root-or-home blocks a recursive delete of /, /*, ~, $HOME and ${HOME} (with / and /* forms), and rm-rf-working-directory warns on *, ., .., ./*, ../* and .*; proved by TestRecursiveDeleteOfRootOrHomeBlocks and TestRecursiveDeleteOfTheWorkingDirectoryWarns, watched failing on a scratch archive first."
impact: fix
resolved_by:
  commit: "e76c78fb6b02768be6cf0e451b972a90292c3736"
---

The guard's bundled registry (internal/core/guard/defaults/guard.json) has one rm entry, rm-rf-after-cd-chain, so a bare `rm -rf *` or `rm -rf /` checked with `abcd guard check` returns allow and exit 0, while the guard chapter's headline promises the catch. Found by the v0.11.1 crosscheck (x-047); registry unchanged since v0.11.0.

## Deferral 2026-09-28

Deferred past v0.11.0: Found by the v0.11.1 release gate after the content commit; the gap already shipped in v0.11.0 (registry unchanged), and a guard-registry change needs its own tests and a security review, so it is fixed in the first lane after the v0.11.1 tag rather than re-rolling this cut

## Grounds

- pursued: a bare recursive delete of the root or the home directory is refused in every flag spelling and behind every launcher the registry steps over, while a delete of a named directory stays an allow; a shape such as rm -rf / or sudo rm -fr ~ returning allow, or rm -rf ./build warning, would show it wrong
