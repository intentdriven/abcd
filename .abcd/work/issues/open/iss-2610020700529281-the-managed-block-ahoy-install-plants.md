---
schema_version: 1
id: "iss-2610020700529281"
slug: "the-managed-block-ahoy-install-plants"
severity: "minor"
category: "ux"
source: "managed-repo"
found_during: "peer report: ahoy install --adopt on a private consumer repo (abcd v0.9.0), reproduced at 7fb52a6b5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/defaults/claude-md-marker-block.md"
remedy: "Plant a short pointer variant in adopted repositories (a few ASCII lines naming the loader, abcd rules and .abcd/rules.json, no repository-internal links or record ids), keep the long text in abcd's own AGENTS.md or behind abcd rules, and place it after the repository's opening text; test: the planted block in a fresh fixture repo has no link that fails to resolve and no non-ASCII byte."
---

The managed block ahoy install plants into an adopted repository's conventions file when a docs target is chosen (internal/core/ahoy/defaults/claude-md-marker-block.md) is abcd's own rule-loader section copied verbatim, and it breaks the repositories that receive it. (a) It ends 'For internals see .abcd/development/brief/05-internals/03-configuration.md' and says OPINIONS points at .abcd/development/principles/; both exist only in abcd's own repository, so they dangle in every adopted one. (b) At tip it is 98 lines with em dashes on 10 of them, so a repository whose own AGENTS.md rules bound file length, ban em dashes or require every link to resolve fails its own gate on the block. (c) Its header says /abcd:ahoy 'silently overwrites this block on drift (per itd-3)', citing an abcd-internal record id, so the adopter cannot trim it to fit. (d) It is inserted directly under the H1, ahead of the repository's own opening text. This record is intent-shaped: the ask is a capability, a short pointer variant of the block for adopted repositories (a few ASCII lines, no repository-internal links or record ids) or keeping the long text where consumer repositories never commit it; routing it to an intent is the product thinker's call. Related: iss-2609110944498549 (resolved, the default docs target is skip) and iss-2608210934566220 (the injected OPINIONS pointers dangle in managed repositories).

## Evidence 2026-10-03

A second report from the same user test of a private consumer repository (relayed by the product thinker): the marker block grew to 98 lines in each of CLAUDE.md and AGENTS.md, so the block is planted twice, once per conventions file, and both copies still point to abcd's principles folder and abcd's own configuration chapter, neither of which exists in that repository. The double planting ties this record to itd-2610030814013772 (one conventions file, AGENTS.md): once abcd writes AGENTS.md alone, the block has one home.

## Evidence 2026-10-04 (a downstream lab)

A downstream private project's brief-authoring lab reported both halves of this record. (a) The planted block's closing pointer at `.abcd/development/brief/05-internals/03-configuration.md` does not merely dangle there: that project keeps its own brief at the same path, so the pointer resolves to the project's own configuration chapter, which says nothing about abcd's loader. (b) The block is about ninety lines of loader internals loaded into every session; the lab proposed the remedy this record already names, a short pointer at `abcd rules` in place of the long text. At tip 57d5ec9fa an install with `--docs-target agents_md` into a scratch repository holding only a README plants an AGENTS.md that is the block alone, 104 lines, still carrying both the principles and the configuration-chapter pointers.
