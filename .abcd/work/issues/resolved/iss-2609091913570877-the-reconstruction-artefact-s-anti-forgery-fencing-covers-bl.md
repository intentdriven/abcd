---
schema_version: 1
id: "iss-2609091913570877"
slug: "the-reconstruction-artefact-s-anti-forgery-fencing-covers-bl"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "fidelity audit of the reconstruction intent"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/reconstruct_render.go"
resolution: "Every scalar the artefact writes outside a fence now goes through the repository's canonical one-line cleaner, in a code span whose delimiter run outruns any backtick inside it, and pipe-escaped in the timeline table. The sweep found nine sites rather than the four first reported: two more are transcript-controlled, an unknown block's type and the tool-call identifier the spawn point is recovered from, and the rest are record-derived where a line break cannot survive the parser but a backtick or a pipe can. Sites already safe were routed through the cleaner too, because a per-site judgement about which values are dangerous is what let the gap open. The reader-facing guide now states the invariant that ships and does not depend on the reader classifying anything: every line of the document begins with words the document chose, and nothing quoted from the session can begin one."
impact: fix
resolved_by:
  intent: "itd-2609091718595846"
  spec: "spc-2609091722269727"
---

The reconstruction artefact's anti-forgery fencing covers block content but not block metadata, so the forgery class it was built to close is still open. A tool call's name, a tool result's identifier and a turn's model name are formatted directly into the document outside any fence, with no guard on their contents: the only helper applied to them substitutes a dash for the empty string. A transcript carrying a line break inside any of those fields therefore emits lines of the document itself, and the lines it can emit include an agent section heading, a turn heading and a join marker, which are exactly the shapes the earlier fix set out to make unforgeable. The regression test plants its payload only in a text block, so nothing exercises the metadata path and the gap passed unnoticed. The consequence falls on the reader the artefact is written for: the guide it carries tells that reader everything inside a fence is something somebody said and everything outside one is the document, and that is not the rule the renderer implements. The fix is to sanitise every scalar interpolated outside a fence rather than to fence more, since these are single-line labels, and to extend the regression test to each of them. Until then the document states a guarantee it does not hold, which is worse than stating none.

## Grounds

- pursued: we expect containment to be the whole answer here, because these are single-line labels whose only power is to leave their line, so stripping the line break removes the forgery and escaping the backtick and pipe removes the quieter falsehoods of escaping a code span and shifting a table's cells; it is shown wrong if a value can forge a line by some route that is not a line break, or if the cleaner's own rewrites make a label misrepresent what the session actually contained
