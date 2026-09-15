---
schema_version: 1
id: "iss-2608301206073609"
slug: "hidden-runes-reach-committed-records-verbatim-because-termsa"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "itd-179-round-2-security"
found_at: "internal/termsafe/termsafe.go"
---

hidden runes reach committed records verbatim because termsafe.EncodeHiddenRunes is not applied at the record-write boundary

Found by the round-2 adversarial security review of build/itd-179, and
recorded rather than chased: the class REPRODUCES ON MAIN.

A bidi override, zero-width space, C1 control or DEL typed into a capture body,
a `wontfix_reason` or an `abcd intent "<text>"` headline lands in the committed
record verbatim today. The branch adds two more instances of the class
(`--grounds` on the ledger side and on the intent side, where the Markdown
writer has no equivalent of `yamlScalar`'s below-0x20 guard) but does not
create it.

The repo already owns the canonical primitive: `internal/termsafe/termsafe.go`
declares itself "the one canonical sanitiser ... before it is written to a
terminal or a human report", and `EncodeHiddenRunes` carries a test saying a
caller can no longer smuggle a terminal escape or a Trojan-Source reorder into
a JSON surface OR A COMMITTED RECORD. It is called at two boundaries only
(`internal/core/cite/fetch.go:212`, `internal/core/memory/ingest.go:694`). It
landed for iss-359, category security, whose remedy was to percent-encode
surviving non-printing runes at the boundary.

So this is a one-canonical-primitive gap, not a missing primitive: the record
write boundary does not call the sanitiser the repo built for it.

FOR THE FACILITATOR: closing this is a separate change with its own blast
radius (every record-writing path), which is why it is captured open rather
than folded into itd-179. Note the interaction: applying EncodeHiddenRunes at
the grounds boundary would also close the invisibility half of
iss-2608301206034359.
