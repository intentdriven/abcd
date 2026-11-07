---
schema_version: 1
id: "iss-2611060810127057"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/positioning/config.go"
remedy: "Make the fallback locators capture the whole paragraph, or limit the chapter's claim to the multi-line locators."
---

`.abcd/development/brief/04-surfaces/19-identity.md:92-94` says comparison folds line wrapping, so a tagline wrapped across two lines is not drift. The comparison sees only what the locator captured, and the fallback locators for `README.md` and `AGENTS.md` (`internal/core/positioning/config.go:120` and `internal/core/positioning/config.go:143`) capture one line. A plain paragraph wrapped across two lines under the top heading is reported `drifted` in both files, showing only its first line, and `abcd identity render` proposes replacing that line with the whole tagline while keeping the second, which would duplicate its text. The claim holds only for the multi-line locators: a `<p>` strapline and the paragraph after the abcd marker block.
