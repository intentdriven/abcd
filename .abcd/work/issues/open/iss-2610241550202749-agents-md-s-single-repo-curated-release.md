---
schema_version: 1
id: "iss-2610241550202749"
slug: "agents-md-s-single-repo-curated-release"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b12 reading iss-2608210934566220"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
remedy: "Reword the four passages to the channel truth: .abcd/** is in every repository checkout and release source archive, and never in the plugin archive a marketplace install receives or in the released binaries, because the launch bundle that renders that archive denies it on every cut; proven by a docs test that fails while any of the four still says marketplace installs carry .abcd/** or that the deny has not run on a cut release."
refines: [iss-183]
---

AGENTS.md's Single repo, curated release boundary, .abcd/work/CONTEXT.md:44-47, .abcd/README.md:4 and .abcd/development/README.md:3-5 still say .abcd/** is present in marketplace installs and that the launch bundler's structural deny has yet to run on a cut release. Since adr-2609231048308186 the marketplace catalog pins each release's plugin archive (.claude-plugin/marketplace.json names abcd-plugin-v0.15.1.zip), and RenderPluginArchive renders that archive through RenderPayload and the bundle's structural deny of .abcd/** (internal/core/launch/archive.go:116-132, internal/core/launch/render.go RenderPayload, internal/core/launch/bundle.go:8-10), so a marketplace install now carries no .abcd/ tree and the deny runs on every cut.
