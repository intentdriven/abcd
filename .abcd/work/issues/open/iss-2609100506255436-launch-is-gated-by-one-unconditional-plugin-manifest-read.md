---
schema_version: 1
id: "iss-2609100506255436"
slug: "launch-is-gated-by-one-unconditional-plugin-manifest-read"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-07/08; re-filed into abcd 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/installsurface.go"
---

Sharpens iss-2608270559313719, which reports that abcd's release flow cannot serve a managed repo that is not a plugin. That record lists the symptoms; this one names the single line that causes most of them, and shows that abcd already solved the neighbouring problem, so the fix has a precedent inside the same file.

`ResolveInstallSurface` reads `.claude-plugin/plugin.json` unconditionally, and `readManifest` returns an error when the file cannot be READ, not only when it cannot be parsed. A repo that ships something other than a plugin has no such file, so resolution fails before anything else runs. Observed as `abcd changelog` refusing with "reading .claude-plugin/plugin.json: no such file or directory" in a managed repository whose artefact is a platform application bundle.

The precedent is in the same file, eight lines above the failing read. The constant's own comment says: "Unlike the VERSION location (which version-location.json makes negotiable per adr-19), the manifest's own path is fixed by the harness's discovery rule — a plugin.json anywhere else is not found at all". The reasoning for the fixed PATH is sound: the harness really does discover a plugin manifest at one location only. But it answers a different question from the one that bites. Where the manifest lives and whether the artefact has one are separate facts, and adr-19 already established that a release-shaped fact can be declared per repo rather than assumed.

So the defect is not that the path is a constant. It is that a MISSING manifest is treated as a broken payload rather than as "this artefact is not a plugin": one unconditional read, sitting next to a sibling problem abcd chose to make negotiable.

What this costs a non-plugin repo, in the order a maintainer meets it. `launch --dry-run`, `changelog` and `launch ship` all refuse, so the derived changelog never runs; the repo falls back to a hand-written CHANGELOG; and the convention that a user-facing change is accompanied by a RECORD rather than hand-written release prose becomes unenforceable exactly where it was meant to bind. The release still ships, because the repo carries its own tag-driven workflow, but abcd's release gates are absent from it, silently, and nothing says so at release time.

Needed: let a repo declare its artefact kind, or treat an absent plugin manifest as an absent plugin rather than an unreadable payload, so that `InstallSurface` carries no plugin name instead of failing. A repo that ships a binary, an application bundle or a library could then use the derived changelog, the payload scan and the release gates, which is the part of `launch` that has nothing to do with being a plugin.

Related, filed separately: the same repo's adoption gap (no launch payload config, no version location for a non-plugin artefact, `launch scaffold` writing a generic Linux Go workflow over a platform-specific one).
