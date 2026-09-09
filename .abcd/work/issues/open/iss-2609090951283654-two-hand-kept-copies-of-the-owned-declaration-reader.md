---
schema_version: 1
id: "iss-2609090951283654"
slug: "two-hand-kept-copies-of-the-owned-declaration-reader"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/root.go"
---

The user-scope declaration reader now exists twice, hand-kept: trustedRootDeclared in the rules resolver and localDeclared in the transcript-store locator run the same five checks in the same order, an lstat for a regular file, a refusal of group or world write, a requirement that the caller own it, a read through the guarded reader under a byte cap, and a comparison of each absolute entry against the target in both its written and its symlink-resolved spelling under the platform fold. Each also carries its own ignored-declaration renderer and its own verbatim copy of resolveOrClean. They have already diverged in one place: the rules copy owns a local ownership seam so a test can force a foreign uid, and the history copy calls the canonical lookup directly, so the refusal branch is provable in one package and not in the other. It matters because this is a trust boundary whose next hardening, a caller-owned parent requirement or a same-file re-check between the ownership stat and the read, will land in whichever copy the fixer happens to be looking at, and the omission compiles green in both. Fix direction: lift the reader into one primitive taking the declaration location and the value to match and returning the verdict plus the ignored-declaration reason, and have both packages call it. Detector: a change to the declaration reader checks must be visible to both callers through one definition, so a test that removes a check fails in both packages.
