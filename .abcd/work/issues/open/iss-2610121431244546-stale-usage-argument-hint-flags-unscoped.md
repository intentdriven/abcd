---
schema_version: 1
id: "iss-2610121431244546"
slug: "stale-usage-argument-hint-flags-unscoped"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "ruthless review of fix/stale-usage-notes, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/staleusage.go"
remedy: "Count a page's argument-hint front matter as a usage block for the typed sub-verb when that page documents one sub-verb (or the hint names it), so a flag listed there is documented for it; test: a release-stamped build outside the plugin root given a flag only capture.md's argument-hint lists gets the stale-copy note, watched fail first."
---

Once the fix for iss-2610120921497084 scopes the stale-usage flag match to fenced synopsis lines (internal/surface/cli/staleusage.go, synopsisCarriesFlag), a flag a command page documents only in its argument-hint front matter or in inline-backtick prose no longer counts as documented. About 60 flags across commands/*.md appear only there (capture resolve's [--commit <sha>], build's --release and --discard, ahoy install --yes), so a release PATH copy that predates one of them now prints the plain unknown-flag error and withholds the true stale-copy note. Fails safe (silence, never a false claim). Found by the ruthless review of fix/stale-usage-notes, 2026-10-12. <!-- record-lint: forward-looking -->
