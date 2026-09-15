---
schema_version: 1
id: "iss-2608301421380392"
slug: "the-raw-heading-soft-bound-does-not-match-a-crlf-blank-line"
severity: "major"
category: "security"
source: "user-observation"
found_during: "itd-183-round-10-ruthless"
found_at: "internal/core/reading/project.go"
resolution: "rawHeadingBoundRe's blank-line alternative becomes \\n[ \\t\\r]*\\n, so a CRLF blank line bounds a raw heading element as an LF one does. rawHeadingTitleEnds now reports whether any bound was found, and verifyRedaction refuses an opener that reaches the end of the document with neither a hard nor a soft bound as 'a raw heading element that is never closed'. The quadratic cost of the title read and the linearity test's name stay with iss-2608301421382564."
impact: fix
resolved_by:
  intent: "itd-194"
  spec: "spc-2609021003136831"
---

the raw heading soft bound does not match a CRLF blank line so an unclosed heading in a CRLF file reads its title over the rest of the document and travels

Found by the round-10 adversarial ruthless review of build/itd-183. NOT
introduced by this branch -- `rawHeadingBoundRe` carries the same blank-line
alternative on the round-9 parent and before it. Recorded rather than chased
under the ruling that round 10 closes only what the branch introduced.

The soft bound on a raw heading's text is `\n[ \t]*\n`, which a CRLF blank line
does not match: its bytes are `\r\n\r\n`, and the `\r` falls outside the class.
So an element that is never closed has no bound at all, its title is read over
the rest of the document, and the slug names something else:

```
<h2>Audit Notes<CR><LF>
<CR><LF>
<sentinel><CR><LF>
```

Verified admitted; the byte-identical document with LF line endings refuses.

This is the same class round 10 closed for YAML keys -- a carriage return is
whitespace to every reader of the file and is not in the scan's own class -- and
it stays open on the heading side. Remedy: `\n[ \t\r]*\n`, or a normalisation
of line endings before the heading scan. Seed material for the exclusion floor's
own intent.

## Grounds

- pursued: we expect a carriage return in the blank-line class, and a refusal for an opener with no bound at all, to stop the title being read over the remainder in either line ending, because the soft bound is the sole bound an unclosed element has; a CRLF document whose excluded heading still travels, or an unbounded opener that still assembles, would show it wrong
