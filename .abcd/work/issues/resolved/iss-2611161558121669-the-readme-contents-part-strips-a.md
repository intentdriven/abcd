---
schema_version: 1
id: "iss-2611161558121669"
slug: "the-readme-contents-part-strips-a"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "correctness review of the iss-2610112021163993 fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/readme_derive.go"
remedy: "Hold each backslash escape outside raw HTML before the tag and link strippers run, reading the heading left to right so a backslash inside a tag stays literal, and restore the escapes before the link text is escaped; test an escaped tag, an escaped link and a link with an escaped bracket in it."
resolution: "headingText now holds each backslash escape outside raw HTML while the tag and link patterns run, reading the heading left to right, and restores them before the link text is escaped, so an escaped tag or link is listed as the heading renders it and an escaped bracket inside a link no longer cuts it short; TestContentsKeepsAHeadingsEscapedMarkupAsText holds it."
impact: fix
---

The README contents part strips a heading's escaped tag or link as if it were markup: headingText (internal/core/ahoy/readme_derive.go) runs inlineTagRe and inlineLinkRe over the heading with its backslash escapes still in place, so a heading written with an escaped angle bracket before b, a closing angle bracket and then emphasised x renders the literal tag and emphasis, while the list drops the tag and keeps only the escaped asterisk, rendering a different text. An escaped bracket before a link's text is stripped the same way, and an escaped closing bracket or parenthesis inside a link cuts the link short.

## Grounds

- pursued: a heading with an escaped tag or link is listed with the text it renders and an anchor the forge would make; a contents line that drops escaped markup would show it wrong
