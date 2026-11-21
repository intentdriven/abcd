---
id: dcn-2611202018389685
slug: a-tool-list-s-reason-is-at-most-160
date: 2026-11-20
---
2026-11-20 — A tool list's reason is at most 160 characters, drawn only from letters, digits, the space and the punctuation .,;:!?()[]-_+=%&#@*, with no combining marks (itd-2611041815429986, step 1). Grounds: the needs-step1 Opus review accepted the limit and the allowlist but found the claim 'letters of any script' overstated, since unicode.IsLetter already refuses every combining mark; it advised against admitting combining marks because U+0334-U+0338 and U+20D2 draw a slash over a letter, which would let a visual path separator into a reason that must not carry a path to type (adr-2611041815445203 rule 2). The simpler wording needs no new dependency; the cost is that an abugida script such as Devanagari cannot be written in a reason. Refs itd-2611041815429986.
