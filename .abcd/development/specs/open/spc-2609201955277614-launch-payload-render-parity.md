---
id: spc-2609201955277614
slug: launch-payload-render-parity
intent: itd-66
origin: researcher-authored
production_mode: hand-written
---
# launch-payload-render-parity

## Summary

This is a REMAINDER spec. A review on 2026-09-20 against the v0.9.0 tree found
the intent about half delivered; the product thinker ruled that the delivered
half is recorded on the intent (its `## Delivery Status` section) and this spec
scopes only what is still open.

## Scope

Two pieces:

1. **File-level parity diff.** The rendered payload is compared with the
   previous release's payload (the tag the cut derives from): every path
   added, changed or removed is listed with its digest, in the preview and in
   the cut's receipt. With no previous tag every path is reported as added and
   the report says why; a baseline that cannot be read is a named refusal, not
   an empty diff. Today only a record-level diff exists (`release/emit.go`,
   added and removed surfaces), never files.
2. **Deep smoke tier.** For every command page and skill in the rendered
   payload, the tier renders its help and frontmatter in an isolated subprocess
   rooted at the rendered tree, so a page that resolves on disk but fails to
   load is caught before the tag. The original criterion's Python-import clause
   is moot (nothing shipped imports) and is recorded as such on the intent.

## Out of scope

The render itself, the structural exclusion of the record namespace, the
symlink defence and the no-residue guarantee are delivered and cited on the
intent.

## Approach

The parity diff reads the previous payload from the tag's release asset when
present and from a fresh render at that tag otherwise, so it never needs a
stored baseline file (adr-31 keeps the version derived, so there is no
baseline to seed). The deep tier is a second level of `internal/core/launch/
smoke.go`, opt-in by flag in the preview and always-on in the cut.

