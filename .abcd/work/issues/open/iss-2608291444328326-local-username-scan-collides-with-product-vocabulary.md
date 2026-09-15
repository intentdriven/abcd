---
schema_version: 1
id: "iss-2608291444328326"
slug: "local-username-scan-collides-with-product-vocabulary"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "v0.6.8 release cut 2026-08-29"
found_at: "internal/adapter/scanner/identity.go"
---

The launch payload's `local_username` identity rule takes the username from the last segment of HOME and hard-fails every case-insensitive occurrence of it in the payload. On a machine whose account name is an ordinary word that is also product vocabulary (an account named with the same three-letter word as the `ahoy install` track-latest flag and the "-shim" install mode it names, which appears in commands/ahoy.md, commands/update.md, commands/launch.md and docs/reference/cli/commands.md), `TestPayloadTreeImplementationsResolveIdentically` and `TestRenderPayloadLeavesSourceTreeUnversioned` fail on a pristine main (9 hard-fail findings at e6efe28f), and with them `make preflight` and the pre-push hook, so the v0.6.8 release branch could not be pushed from that machine until HOME was pointed at a symlink alias of the real home directory for the push. CI, which runs under a different account name, passes.

The rule needs a notion of a username that is too common or too short to be identifying (a dictionary or length floor, or an explicit allowlist in the scanner config), and the finding should say which identity source matched, so a maintainer can tell a collision from a leak. The capture redactor has the same collision: it blanked the flag name out of this record's first draft.
