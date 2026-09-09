---
schema_version: 1
id: "iss-2609090947359464"
slug: "the-ledger-root-resolver-is-the-unhardened-sibling-of-the-rules-root"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/roots.go"
---

discoverRepoRoot is the unhardened twin of the rules-root resolver that was hardened in the same batch. Its git call is properly isolated, and its own comment explains why: an inherited GIT_WORK_TREE or GIT_DIR would redirect discovery at a different tree. The fallback beneath that call has neither guard the sibling grew. Where git will not answer, the loop walks upward accepting any directory whose git entry merely exists, with no shape check of the kind plausibleRepository performs and no ownership check of the kind foreignOwnerRefusal performs, so an empty directory named for the marker in a shared ancestor, or a real repository another uid laid there, would bound the ledger root exactly as it bounded the rules root before that fix.

CORRECTED 2026-09-09, major to minor, on reachability. This record first asserted a live harm path, that a capture made beneath such a plant could be written with the attacker's redaction rules and into the attacker's tree, and that is false. What stops it is reachability rather than the walk: every front door supplies an explicit repo root, so the fallback branch is dead in shipped code. The CLI hands its working directory verbatim to every capture request and the reading verbs resolve the toplevel themselves before they call, so nothing shipped leaves the root empty for the discovery helper to answer. Reproduced against the plant the first draft described: with an empty marker directory and a populated ledger planted in a shared ancestor, the status verb run from a directory beneath it reported open 0 rather than reading the planted ledger. The redaction limb fell with it, because the scanner is built from the root the surface passed, which is the working directory and never the planted one.

What remains, and why this is still worth a record: the gap in the walk is real, and the tree treats this function as the fixed exemplar of repo-root discovery, cited by name in the resolution of iss-311 as the shape two other resolvers were corrected to match, so the next surface that leaves the root empty inherits an unhardened walk with nothing saying so. That the branch is unreachable is a property of today's callers rather than of the function, and no test pins it. Fix direction: route the fallback through the same shape and ownership gate the rules resolver uses, reusing plausibleRepository and the ownership refusal rather than restating them, and honour the same explicit opt-in so a container bind mount and a shared CI checkout keep working; or delete the walk and refuse outright when git cannot answer. Detector: with git unable to answer, a walk reaching a directory that carries only an empty git marker must resolve no repo root, and a marker root owned by another uid must be refused unless the caller has declared it. The separate defect that the front doors never call this helper at all is recorded as iss-2609090951291524.
