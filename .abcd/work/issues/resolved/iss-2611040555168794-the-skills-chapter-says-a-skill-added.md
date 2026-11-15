---
schema_version: 1
id: "iss-2611040555168794"
slug: "the-skills-chapter-says-a-skill-added"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/08-skills.md"
remedy: "Name the test as the guard for the payload step and replace the 'missing from the release' consequence with the failure it actually gives."
resolution: "The skills chapter now names TestBundleShipsEveryPluginSurface as the guard for the payload step and says a skill left out of the include list fails that test in preflight and CI, rather than being missing from the release; TestSkillsChapterNamesThePayloadGuard holds the chapter to it."
impact: internal
resolved_by:
  commit: "e910140f907ef9ca622bb254c4cf58294f43f290"
---

The skills chapter says a skill added without adding the skills directory to the payload's include list passes the record gate and is missing from the release (.abcd/development/brief/05-internals/08-skills.md:120-124). TestBundleShipsEveryPluginSurface (internal/core/launch/payload_completeness_test.go:40) fails on a skills/ directory that is neither in the includes nor in surfaceExcludeReasons, and it runs in `go test ./...`, so in preflight and CI. Run against a copy holding skills/foo/SKILL.md with the includes unchanged, it failed. Such a skill cannot land by the normal route.

## Grounds

- pursued: the chapter names the test that actually guards the payload step and the failure it gives; shown wrong if TestBundleShipsEveryPluginSurface passed on a tree with skills/foo/SKILL.md and unchanged includes, or did not run in go test ./...
