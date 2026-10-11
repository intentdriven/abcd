---
schema_version: 1
id: "iss-106"
slug: "gl002-stripinlinecode-restores-the-whole"
severity: "minor"
impact: fix
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "stripInlineCode now blanks matched backtick pairs only; a trailing unpaired backtick and its tail stay literal so earlier closed spans remain masked (double-backtick spans left out of scope). Detector: TestStripInlineCodePairedSpanBeforeStrayBacktick + TestForbiddenSynonymsClosedSpanThenStrayBacktick."
---

GL002 stripInlineCode restores the whole line on an unpaired trailing backtick (and mishandles double-backtick spans), so a closed code span containing an enforced synonym plus a later stray backtick fires a spurious blocker — fails toward over-flagging, never misses drift; found by burst-5 correctness review, not present in current corpus