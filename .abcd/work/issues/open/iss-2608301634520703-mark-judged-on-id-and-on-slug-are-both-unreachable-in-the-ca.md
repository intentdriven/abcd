---
schema_version: 1
id: "iss-2608301634520703"
slug: "mark-judged-on-id-and-on-slug-are-both-unreachable-in-the-ca"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-189-round-4-builder"
found_at: "internal/core/lint/schema.go"
---

mark judged on id and on slug are both unreachable in the case they guard so removing either alone repeats the shape the round was convened for

Reported by the round-4 builder, which deliberately did NOT act on it and said
so. Recorded because the reasoning should outlive the round rather than sit
only in a resolution stamp.

`mark(judged, "slug")` at schema.go:705 is redundant: the content legs return
early on an empty value, and `judged` is consulted only for a present-but-empty
one, so the mark cannot be reached in the case it guards. `mark(judged, "id")`
at :630 is redundant on identical grounds.

Removing slug alone would reproduce exactly the shape this round was convened
for, one instance removed and its sibling left, which is
iss-2608301519254240's opening sentence. Removing both leaves `judged` unused
in two signatures and contradicts the protocol stated at the call site, that
the content legs run first and mark what they spoke about. Neither move is
obviously right, which is why this is a record rather than a commit.

The decision wants making once, for both, with the call-site protocol either
honoured or rewritten. It is cosmetic in every case and blocks nothing.
