---
schema_version: 1
id: "iss-2608301251385581"
slug: "the-comment-mask-suppresses-heading-discovery-so-an-excluded"
severity: "critical"
category: "security"
source: "user-observation"
found_during: "itd-183-round-9-ruthless"
found_at: "internal/core/reading/project.go"
resolution: "The heading scan reads the document masked AND unmasked and refuses on either, so the markup mask can only add a reading and never hide one."
impact: fix
resolved_by:
  intent: "itd-183"
---

the comment mask suppresses heading discovery so an excluded heading wholly inside an HTML comment is admitted though the file still carries it

Found by the round-9 adversarial RUTHLESS review of build/itd-183.
REGRESSION INTRODUCED BY 2225d6cb. **Not found by the parallel security review,
and NOT closed by its remedy** -- this comment is properly terminated, so
bounding the attribute search leaves it wide open. It needs its own fix.

`maskAngles(out, i+4, i+4+end)` blanks every angle bracket inside a
`<!-- ... -->` span before `rawHeadingOpenRe` runs, so a heading that sits
wholly inside a comment is never discovered:

```
<!-- open

<h2>Audit Notes</h2>

private provenance

close -->
```

No opener is found, the run is admitted, and the section travels. Confirmed
end to end; the parent 044ac6ed refuses.

Why "it's only a comment" is the wrong answer, and the heart of this record:
**the blind reader receives raw markdown, not a rendered page.** The charter's
contract is that a file which still CARRIES an excluded key or heading after
redaction refuses the run. A commented-out heading is carried. The mask was
built to stop a comment's angle brackets from being read as STRUCTURE; it
should never have stopped them from being read as CONTENT.

Remedy (class-closing): mask a comment's brackets only for the purpose of
BOUNDING an already-found heading, never for FINDING one -- run
`rawHeadingOpenRe` over the unmasked text and apply the mask only to `rest`
when computing bounds. Minimally: bound the comment mask at a fence boundary
and keep refusing when the comment's own interior names an excluded heading.
