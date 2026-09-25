---
id: spc-23
slug: development-instructions-live-in-the-right-tier-abcd-manages
intent: itd-117
---
# development-instructions-live-in-the-right-tier-abcd-manages

## Summary

spc-23 delivers itd-117: a user-scope rules layer at `~/.abcd/rules.json` that
composes between abcd's plugin-bundled default domains and each repo's in-tree
`.abcd/rules.json`. Machine-wide house conventions are declared once; a repo
override still wins; a machine without the file behaves exactly as today.

## Scope

- **Core** (`internal/core/rules`): `Load` gains a user-scope read and one more
  merge step — `Merge(Merge(Defaults(), user), repo)`. The user file passes the
  same trust guards as the repo file, and its layer of origin is recorded so a
  surface can render provenance.
- **Surface**: `abcd rules` shows which layer each domain came from.
- **Not in scope**: any directory-designated workspace or repo-grouping layer.
  There is no `workspaces.json` and none is introduced — itd-40's no-grouping
  stance stands untouched. Duplication
  detection between layers and migration of harness memory are deferred to
  follow-up intents (recorded in itd-117's Open Questions).

## Approach

The existing loader already has every primitive this needs, so the change is
composition rather than new machinery:

- `readGuarded(path, maxRulesFileBytes)` (rules.go) opens read-only with
  `O_NOFOLLOW`, refuses non-regular leaves, and caps at 256 KiB in a single
  open — reused verbatim for the user path, with the `.abcd` symlink
  pre-check applied to `~/.abcd` exactly as it is to the repo's.
- `Merge(base, over)` is per-field with a **sticky kill switch**
  (`out.Disabled = base.Disabled || over.Disabled`). Applying the repo layer
  last therefore satisfies AC3 (repo wins per field) and AC7 (a repo's
  `dormant` state or kill switch survives a user layer that declares the domain
  active) without new merge semantics.
- **Consequence worth recording**: stickiness runs both ways — a user layer
  setting `"disabled": true` silences abcd's rules on that machine and no repo
  can re-enable it. That is the fail-safe direction (a kill switch that can be
  overridden downward is not a kill switch), and it is documented rather than
  worked around.
- Absence is the default: `os.IsNotExist` on the user path yields the bundled
  defaults unchanged, so AC1's byte-identical guarantee falls out of the same
  branch the repo path already uses.
- The user scope's location comes from the existing user-scope resolution
  (`~/.abcd/`), which the brief already blesses for machine-local shared state
  — no new path concept is introduced.

## Acceptance-criteria satisfaction

- **AC1 (absence is free)** — the `os.IsNotExist` branch returns the prior
  result; test asserts a deep-equal ruleset against `Defaults()` and that no
  file or directory is created.
- **AC2 (user overrides bundled)** — `Merge(Defaults(), user)`; test sets one
  field in a bundled domain and asserts the injected value.
- **AC3 (repo overrides user)** — repo layer applied last; test sets the same
  field at all three layers and asserts the repo value.
- **AC4 (custom user domain)** — `Merge` adds unknown domain keys; test
  declares a domain only in the user file and asserts it recall-matches in a
  repo that never mentions it.
- **AC5 (loud refusal)** — the `ELOOP` / `errNotRegular` / `errTooBig` /
  invalid-JSON branches are mirrored for the user path with user-scope-specific
  messages; test asserts each is a hard error, never a silent fallback to
  defaults.
- **AC6 (provenance)** — each domain records its originating layer through the
  merge; `abcd rules` renders it, and the CLI test asserts the label for a
  domain overridden at each level.
- **AC7 (repo suppression wins)** — covered by ordering plus kill-switch
  stickiness; test asserts a repo `dormant` beats a user-active domain, and
  that a repo kill switch suppresses everything.

## Known limitations

`mergeDomain` replaces a domain's fields wholesale — a partial rules-array
override is not expressible, and a third layer makes that grain more visible.
Recorded in itd-117's Open Questions as a known limitation, not fixed here.
