---
id: dcn-2611131012147508
slug: the-docs-lint-s-reporting-uses-below-a
date: 2026-11-13
---
2026-11-13 — The docs lint's reporting uses below a backtick left open take the line-alone result first and add what the whole reading finds (iss-2611130850006749). This corrects dcn-2611130921444350, which said the lint still reports all it reported before: tests hold that for links, fragments, footnote markers and forbidden synonyms, and for the other reporting uses (footnote definitions, cited URLs and DOIs, the baseline rule) it rests on the same code path, the line-alone result being the one the lint read before.
