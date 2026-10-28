---
id: dcn-2610272345401009
slug: build-s-claim-stage-refusal-keeps-naming
date: 2026-10-27
---
2026-10-27 — build's claim-stage refusal keeps naming the --session it was given, and a path to something that exists in the working directory is not withheld at the error boundary. Grounds: the claim refusal is reached only by a session that has already joined the run, so the name it repeats is one the run holds, not arbitrary text, while any other value is refused earlier through the error boundary, which withholds it; changing that refusal would also touch a file another lane holds for unrelated work. An existing repository path is what a footprint or record refusal must name for the caller to see the overlap, and nothing secret is a path that exists in the checkout. A leading part of an argument is withheld only where a refusal visibly cut it (an ellipsis, a closing quote or the end of the line), so a refusal naming a longer path that shares a prefix stays readable. Decided by abcd-0b (technical, delegated).
