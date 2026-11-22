---
id: dcn-2611221728296359
slug: model-matrix-step-8-keeps-the-narrow
date: 2026-11-22
---
2026-11-22 — Model-matrix step 8 keeps the narrow rule for the network screen as its fix: a value is withheld from the screen only when the whole of it is a dotted number, and every other value is read as written. It was kept in place of a redesign of the screen because it closes the review finding by construction and rewrites no value in part, where four rounds of partial rewrites had each failed review. To reverse it, revert 79a8b19a2 and apply the form the reviewer advised.
