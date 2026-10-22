---
schema_version: 1
id: "iss-2610222008081888"
slug: "abcd-build-refuses-an-intent-as-held-by"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "lane D build start of itd-2610150602547271"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/check.go"
remedy: "Count a peer as holding a record only when the peer changed it since its merge-base with the default branch (the record's path or blob differs from the merge-base tree's); keep failing closed when the merge-base or the comparison cannot be read; apply the same rule in the abcd peers holder listing."
resolution: "peers.Report.Holders counts a peer as holding a record only when it holds it in another folder and changed it since its merge-base with the default ref (same path and blob, or for a worktree the same on-disk content, as the merge-base tree is a stale copy); it fails closed when the default ref, the merge-base or the comparison cannot be read. The build's peers check uses it; the not-found refusal keeps every location from Locate."
impact: fix
---

abcd build refuses an intent as held by a peer when the only holders are long-lived local branches cut before the intent left drafts/, which hold the default branch's older copy of the record unchanged since their merge-base. On 2026-10-22, build itd-2610150602547271 was refused (exit 3) naming feat/hosted-sites-design, feat/hosted-sites-setup and fix/fence-day-lease as holding it in drafts/; none of the three changed the record since its merge-base with origin/main (1183-1185 commits behind). peersCheck in internal/core/implement/loop/check.go counts every Location in a bucket other than this checkout's, so any stale branch refuses every intent planned after it was cut; with ~190 worktrees on one machine this blocks most builds and build next.
