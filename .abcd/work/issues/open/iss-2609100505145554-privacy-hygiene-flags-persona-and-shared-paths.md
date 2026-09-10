---
schema_version: 1
id: "iss-2609100505145554"
slug: "privacy-hygiene-flags-persona-and-shared-paths"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-07/08; re-filed into abcd 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (lint, privacy-hygiene rule)"
---

The `privacy-hygiene` rule flags persona-derived absolute paths, which the same conventions require examples to use, so a repo that follows the convention cannot pass the lint. In a managed repository it produced 48 findings, 33 of them errors, and every single one was benign. A later run of the same repository's lint reported between 147 and 213 privacy errors, all of the same two classes.

What was actually flagged: persona home paths under the three names the conventions themselves mandate (Alice, Bob, Carol) in test fixtures and documentation, plus the group-readable shared-directory root the product creates and therefore has to name in its own comments, tests and install docs, plus a set of classifier fixtures. No real username appears anywhere in the tree; the only home directories committed are the personas the conventions ask for.

Why this is worse than noise. The rule's own fix hint already blesses persona material: it says to replace a network identifier with "a reserved documentation value (RFC 5737/3849/2606/7042, or a persona-derived device name)". So personas are understood to be the safe form for one identifier class and not for the other, with no stated reason. The result is dozens of errors a maintainer must learn to ignore, which is exactly how a real leak gets waved through later: the rule that cries wolf on a persona home path is the rule nobody reads when it finally names a real one. It also puts the lint permanently red on a repo that is, on this rule, clean.

The cost is measurable in the autonomous run this was observed in. Every worker had to be told in its launch prompt to ignore the privacy count, and two of them stopped and asked whether they had broken something. A detector that is red at baseline is a detector nobody reads, and a per-worker instruction to ignore a specific gate is a standing invitation to ignore the next one.

Needed: teach `privacy-hygiene` the persona names the conventions already fix, so a path under one of them is not a finding, the same way a persona-derived device name is not. A shared-directory system root is likewise not a personal path and is frequently a real product path that documentation has to state. Failing that, the rule should say in its message that the escape for a deliberately illustrative persona path is `abcd-lint:allow`, and the convention should say that a repo using persona paths is expected to carry that marker on every one. That is a worse answer, because it means annotating every example the conventions asked for.

The shared-root half was reported upstream once before, in 2026-07, and had not landed in the release the run was using.
