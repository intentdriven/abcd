---
schema_version: 1
id: "iss-85"
slug: "managed-attribution-config"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "2026-07-10 prepare-this-repo skill grilling"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed B): Is managed attribution setup owned by the configuration layer through itd-91 (M4), or by itd-131's identity pin (M2)?"
remedy: "Waits on ruling B (owner: itd-91's configuration layer or itd-131's identity pin): if itd-91, plan a declared attribution preference under .abcd/config (trailer form such as Assisted-by, or none) that abcd writes into a prepare-commit-msg hook and checks in the attribution lint, retiring the private template, proven by a test that installs into a temporary repository and commits; if itd-131, have ahoy install write the same hook beside the identity pin, with the same test."
---

Managed attribution config: AI-disclosure attribution (prepare-commit-msg hook adding the kernel-style trailer, hooksPath wiring, AGENTS.md section) is installed by the interim prepare-this-repo skill from a private template directory. No intent covers the configuration layer owning attribution setup in managed repos; it should, so the per-repo attribution policy is declared and enforced by the tool rather than a hand-copied hook.

## Remedy grounds (2026-09-29)

- The Linux kernel's coding-assistants guidance fixes the Assisted-by trailer form and forbids an agent adding Signed-off-by (https://docs.kernel.org/process/coding-assistants.html, consulted 2026-09-29), the convention this repository follows, so the preference is a value a hook composes rather than prose to copy; itd-131's own text puts attribution with itd-91, which the ruling may confirm. Fedora's policy page could not be read on 2026-09-29.
- Rejected: the hand-copied hook from a private template, which does not travel to managed repositories.
