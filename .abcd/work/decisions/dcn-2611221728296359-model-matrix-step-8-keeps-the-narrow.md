---
id: dcn-2611221728296359
slug: model-matrix-step-8-keeps-the-narrow
date: 2026-11-22
---
2026-11-22 — Model-matrix step 8 keeps the narrow rule for versions on the network screen as its fix: a version is withheld from the screen only when the whole of it is a dotted number, and every other version is read as written. Hex digests and the field names abcd builds stay withheld as before. The rule was kept in place of a redesign of the screen because it closes the review finding by construction and rewrites no value in part, where four rounds of partial rewrites had each failed review. The alternative the review advised keeps the last part of a dotted number in the screened copy; no other record holds it. To reverse this decision, revert 79a8b19a2 and 3a24b096b together, since the second pins the narrow rule in a test and in comments, and build that alternative.
