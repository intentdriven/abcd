---
id: dcn-2611122004497694
slug: the-page-readers-the-spec-readiness
date: 2026-11-12
---
2026-11-12 — The page readers (the spec readiness filter, the scope-condition and hand-audit readers and the prose-citation rule) blank an inline code span only on a line that no earlier line of its stretch leaves a backtick run open on, a stretch being the lines with no blank line between them, and read the rest: from a run a line leaves with no closer on that line to the stretch's end they blank no span and open no comment, and a span over a line break is read as prose. Grounds: the tree holds no reference markdown renderer, so a reader that mirrors the page's paragraph and inline rules cannot be proven against the page, and two reviews of such a mirror found lines it blanked that the page shows as prose. Reversal: adopt a markdown renderer as a dependency, on the person's word, and pair spans as it does.
