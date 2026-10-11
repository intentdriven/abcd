---
schema_version: 1
id: "iss-2608241959573830"
slug: "the-repository-moved-organisations-and"
severity: "major"
category: "tech-debt"
source: "user-observation"
found_during: "v0.6.4 release session 2026-08-24"
found_at: "go.mod"
resolution: "go.mod's module path and the 404 import lines in 226 Go files, README.md's badges and install commands, docs/how-to/install.md, docs/README.md, docs/explanation/README.md, CITATION.cff, SECURITY.md, the Makefile ldflags, mkdocs.yml, site-src/install.sh.tmpl and the runnable gh command in the rulesets mirror all name the current organisation. Two test fixtures that asserted the old slug move with it. CHANGELOG.md's entry for the earlier transfer and the historical records under .abcd/ keep the old name, because they state facts about the past."
impact: fix
---

the repository moved organisations and the module path, install commands and support links still name the old one. github.com/<old-org>/abcd appears 468 times across 40+ files: go.mod's module path and the 404 import lines in 226 Go files that follow it, README.md's two badge URLs and its marketplace-add and install one-liner, docs/how-to/install.md, CITATION.cff's repository-code, SECURITY.md's advisory link, and the Makefile ldflags that stamp the version symbol. The runtime surface is already corrected (iss-2608241659573856) because it was breaking fresh installs; this is the rest. TestInstallGuideDocumentsTheInstallAndUpdatePath derives the documented marketplace slug from go.mod's module path, so go.mod, README.md and docs/how-to/install.md must move in one change or the gate fails - which is the detector working. Renaming the module path is a breaking change for any Go importer, judged near-zero cost because abcd is a CLI rather than a library and is pre-1.0. Deliberately out of scope: CHANGELOG.md's entry for the EARLIER transfer to the old organisation, and the historical records under .abcd/, which state facts about the past that rewriting would falsify.