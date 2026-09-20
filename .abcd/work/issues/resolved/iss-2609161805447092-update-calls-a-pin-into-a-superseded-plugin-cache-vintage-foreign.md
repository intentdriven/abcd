---
schema_version: 1
id: "iss-2609161805447092"
slug: "update-calls-a-pin-into-a-superseded-plugin-cache-vintage-foreign"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "v0.9.0 deploy, session abcd-2c"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/update_target.go"
resolution: "A pin into a sibling plugin-cache vintage whose binary still exists now classifies as abcd's own (owned-superseded): the update verb refuses with the honest detail (the vintage the pin points at) and names abcd ahoy install, detection raises a required, resolvable symlink.superseded gap instead of the foreign one, and ahoy install rewrites the pin at the current release instead of returning early. The shape is scoped to a plugin root that is not a git checkout, so a live link beside a source checkout stays foreign."
impact: fix
---

abcd update calls a pinned install foreign when its symlink points into a superseded plugin-cache version. On this machine ~/.local/bin/abcd is the pin abcd ahoy install wrote on 2026-09-07: a symlink into the plugin cache directory of the v0.7.1 vintage (e3696dc524e3). After the plugin moved on, go run ./cmd/abcd update v0.9.0 --yes refused with 'the entry at ~/.local/bin/abcd is not something abcd owns, and abcd never clobbers a binary it does not own' and offered 'remove or rename the occupant' as the remedy. The link target is a published release binary (its digest is the darwin-arm64 line of v0.7.1's checksums.txt) inside abcd's own cache, so nothing about it is foreign: the classifier in internal/core/ahoy/update_target.go only admits a symlink into the CURRENT plugin root as owned, and a link into an older vintage falls through to foreign before the digest check the verb's help promises ever runs. The refusal's detail and remedy are both wrong for this shape; the right remedy is abcd ahoy install, which the owned-dangling case already names, and the honest detail is that the pin points at a superseded vintage. Found while finishing the v0.9.0 deploy on 2026-09-16.

## Grounds

- pursued: the pin was written by abcd into abcd's own cache, so calling it foreign sent the operator to hand-delete their own install; owning the shape gives every surface the same remedy, and the git-checkout gate keeps a developer's deliberate sibling-build link untouched
