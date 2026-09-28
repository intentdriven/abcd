---
schema_version: 1
id: "iss-2608301421381157"
slug: "the-escaped-key-refusal-returns-through-an-error-message-ass"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-183-round-10-ruthless"
found_at: "internal/core/reading/project.go"
resolution: "The escaped-key refusal has its own message: it names the key's raw spelling, the line and the ground (a YAML escape this package does not decode), and asserts neither an excluded key nor an unclosed block. The excluded-key message no longer claims a block shape either; it states that the field reader did not report the key, so the redactor did not remove it."
impact: fix
resolved_by:
  commit: "85f1cff9c45196026a5b5095b4e4b33ae7bd9f2a"
---

the escaped key refusal returns through an error message asserting the block is unclosed and the key excluded when neither need be true

Found by the round-10 adversarial ruthless review of build/itd-183. Pre-existing
-- the message is round-8's, and round 10 reintroduced a second caller of it
when it restored the line-anchored escape refusal.

`excludedKeyInFirstBlock` reports every finding through one message:

```
still carries the excluded key %q at line %d after redaction; the
frontmatter block is not closed the way the field reader expects it
```

For the escape refusals both halves of that are false. A frontmatter line
spelled `"C:\tmp\x": v`, with only `origin` excluded, is refused with a message
naming a key that is not on the exclusion list and a block that is closed
exactly as expected. The real ground is the one `unresolvableFrontmatterShape`
states honestly for its own class: this package will not resolve a YAML escape,
so the escape is the signal and the answer is a refusal rather than a guess.

Cost is comprehensibility, not safety -- the refusal is correct, its stated
reason is not. Remedy: give the escape refusal its own message. Seed material
for the exclusion floor's own intent.

## Grounds

- pursued: every escaped-key refusal, line-anchored or in a flow mapping, states the escape and the line and never claims an excluded key or an unclosed block; a refusal message asserting either of an escape, or an escape admitted, would show it wrong
