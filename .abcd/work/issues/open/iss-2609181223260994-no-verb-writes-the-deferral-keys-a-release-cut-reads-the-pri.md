---
schema_version: 1
id: "iss-2609181223260994"
slug: "no-verb-writes-the-deferral-keys-a-release-cut-reads-the-pri"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "Gropius managed-repo session gropiusllm-56, three release cuts by hand, relayed to abcd-17 on 2026-09-18"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

No verb writes the deferral keys a release cut reads. The principle pre-existing-is-not-a-defence names deferred_after and deferral_reason as the one sanctioned way past the release-cut guard for an open major or critical record, and changelog.GuardFindings reads them, but no capture sub-verb writes them: the ledger verbs are disposition, list, mentions, promote, resolve and wontfix, so a deferral is a hand edit of frontmatter that bypasses the validators every other transition carries (the anchor tag's shape, the reason's presence, the record being open and major). Relayed from the Gropius managed-repo session gropiusllm-56 on 2026-09-18 at v0.9.0, which wrote the two keys into six records by script in one day, across three release cuts. Wanted: abcd capture defer <iss-N> --after vX.Y.Z --reason "…", refusing a record that is not open, a tag that is not the current anchor, and an empty reason, and reporting the deferral in the same envelope shape as resolve. Note the gate that reads the keys lives in the launch verbs, which that repository cannot run (iss-2609061432214212), so the write path and the read path are missing together there; this record is the write half.

**Corroboration (2026-09-19, Gropius session gropiusllm-66, relayed to
abcd-17).** Second session, same repository, same hand path the day after:
a deferral is a `## Deferral <date>` section hand-appended to the record body
plus `deferred_after` and `deferral_reason` inserted into the frontmatter by
sed. The ask is unchanged, `abcd capture defer <iss-N> --after <tag> --reason
"…"`, one-shot and lintable; note the body section is part of the shape the
verb would have to write, not only the two keys.
