---
schema_version: 1
id: "iss-2608301350533102"
slug: "fencemask-spans-the-frontmatter-so-a-fence-delimiter-inside"
severity: "critical"
category: "security"
source: "user-observation"
found_during: "itd-183-round-10-security"
found_at: "internal/core/reading/project.go"
resolution: "The fence mask no longer spans the frontmatter block. verifyRedaction locates the block with firstBlockRange over an unmasked document before any mask is computed, and bodyFenceMask starts the toggle after the block closes, so nothing inside the block can switch the excluded-key scan off. The delimiter itself is refused by unresolvableFrontmatterShape as 'a fence delimiter inside the frontmatter block'."
impact: fix
resolved_by:
  intent: "itd-194"
  spec: "spc-2609021003136831"
---

fenceMask spans the frontmatter so a fence delimiter inside a block scalar switches off the excluded key scan for the rest of the block

Found by the round-10 adversarial security review of build/itd-183. NOT
introduced by this branch -- it is present on the round-9 parent too, and is
recorded rather than chased under the ruling that round 10 closes only what the
branch introduced.

`fenceMask` is computed over the WHOLE document, frontmatter included, so a
fence delimiter written inside a frontmatter block scalar toggles the mask on
and marks the rest of the block as fenced. Both `excludedKeyInFirstBlock` and
`unresolvableFrontmatterShape` skip a fenced line, so every key spelling
`frontmatter.Fields` does not itself report then travels:

```
---
a: |
  ```
"origin": <value>
b: |
  ```
---
```

A YAML reader reads a top-level `origin` there. `Fields` wants an unquoted key
at column 0 and does not report it, the redactor drops nothing, and the refusal
that exists to catch exactly that gap is switched off. Verified leaking on this
round and on the parent, for the quoted spelling and for the indented one.

Candidate remedies: scope `fenceMask` to the body, or refuse a frontmatter block
containing a fence delimiter -- the second is the cheaper and matches how
`unresolvableFrontmatterShape` already answers a construction this package will
not parse. Seed material for the exclusion floor's own intent.

## Grounds

- pursued: we expect locating the frontmatter block before any mask is computed, and refusing a fence delimiter inside it, to close the switch-off path rather than one spelling of it, because the mask can no longer be toggled from a region the key scan is walking; a document whose frontmatter carries a fence and still assembles, or an excluded key surviving inside a block the manifest marks parsed, would show it wrong
