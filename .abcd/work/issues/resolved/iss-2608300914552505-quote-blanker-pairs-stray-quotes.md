---
schema_version: 1
id: "iss-2608300914552505"
slug: "quote-blanker-pairs-stray-quotes"
severity: "major"
category: "security"
source: "impl-review"
found_during: "itd-183 seventh-round security review, 2026-08-30"
found_at: "internal/core/reading/project.go (quotedSpanRe, blankQuoted)"
resolution: "The blanker treated every quote as a scalar opener, so an apostrophe in ordinary prose, a stray quote, or an escaped quote paired with a later one and blanked the excluded key between them — the flow scan then never saw a nested origin and the value travelled under a manifest asserting refusal. A quote opens a scalar only in scalar position: at line start or after a colon, brace, bracket, comma or sequence dash. The double-quoted alternative is escape-aware and the single-quoted one honours the doubled-quote escape, and only the captured scalar is blanked, never the opener that introduced it, because blanking the opener erases the comma the next key is read by. The false positive this blanking exists to prevent stays prevented, escapes and all."
impact: fix
---

The quote blanker treats any quote character as a scalar opener, so a quote inside a plain scalar (an apostrophe in ordinary prose) or an escaped quote inside a double-quoted scalar pairs with a later quote and blanks the excluded key sitting between them; the flow-key scan then never sees a nested origin and the value travels while the manifest asserts refusal — demonstrated end to end on three shapes. Blank only quotes in scalar-opening position (line start, after a colon-space, comma, brace, bracket or dash-space) with escape-aware double-quote matching, or refuse a quote that is not in opening position on any flow line as unresolvable; add the three shapes as tests.
