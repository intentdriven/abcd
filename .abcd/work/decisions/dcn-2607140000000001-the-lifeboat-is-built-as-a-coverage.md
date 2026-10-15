---
id: dcn-2607140000000001
slug: the-lifeboat-is-built-as-a-coverage
date: 2026-07-14
---
2026-07-14 — The lifeboat is built as a COVERAGE EXPERIMENT, not a feature
(adr-35, itd-88/spc-3). Probe before pack: `disembark probe <repo>` produces a
cross-repo coverage aggregate BEFORE a packer exists, because the brief's
structure is an untested assumption and building the packer first assumes the
answer. The headline number is the delta in section coverage between a
rich-record repo and a git-only one — that is what the record is worth, and if
half the brief is permanently blank everywhere, the structure is wrong and we
learned it for one milestone instead of a phase. Phase 6's "depends on every
prior substrate being native" rationale was checked against the BINARY and found
mostly false (spec engine ships; reviews are committed markdown; backgrounding is
a host affordance; the itd-2 host-delegation seam already ships twice — memory's
`--pages-json` and `intent review ingest --verdict-json`). The ONE real
dependency is data, not code: `~/.abcd/` does not exist, `history.Capture` is
called by nothing, and Pass B's corpus cannot be obtained retroactively — the
only permanent, compounding cost on the board, which is why the transcript hook
ships ahead of any lifeboat code. Rejected: building the packer first (assumes
the answer); amending adr-4 in place (two of its three operative claims change —
a replacement, not a clarification).
