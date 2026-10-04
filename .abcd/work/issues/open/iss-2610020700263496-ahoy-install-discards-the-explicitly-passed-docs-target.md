---
schema_version: 1
id: "iss-2610020700263496"
slug: "ahoy-install-discards-the-explicitly-passed-docs-target"
severity: "minor"
category: "bug"
source: "managed-repo"
found_during: "peer report: ahoy install --adopt on a private consumer repo (abcd v0.9.0), reproduced at 7fb52a6b5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
remedy: "Persist the flag-supplied values even when a later prompted value is refused (or validate every flag before the first question); on an out-of-set scan_deep answer print a note naming the answer, the accepted set and --scan-deep, and list config.scan_deep_missing as the remaining gap instead of the flagged ones; test: a piped 'y' to scan_deep with the three flags passed leaves them in config.json and names only scan_deep as remaining."
---

ahoy install discards the explicitly passed --docs-target, --visibility and --oracle-backend values when the scan_deep question is answered out of set, and says neither that it did nor why. stepConfigValues collects every config value and persists them in one write only after the last question is answered; on a private repository with trufflehog on PATH it asks scan_deep, the documented 'yes |' form answers 'y', which is not in true|false, so the step returns nil and nothing is written. .abcd/config.json keeps only attribution and meta; the run ends 'partial' with remaining gaps config.docs_target_missing, config.oracle_backend_missing and config.visibility_missing, the three values the operator passed as flags, never scan_deep; no note says the scan_deep answer was refused; and the receipt's remedy, 'run abcd ahoy install again and answer y to each question', repeats the cause. The neighbouring artefact_kind question treats the same 'y' as its default with a note naming the answer. The bare detection pass cannot preview the question, because config.scan_deep_missing is raised only once visibility=private is persisted, not when it arrives as a flag. Reproduced at tip with: yes | abcd ahoy install --adopt --docs-target agents_md --visibility private --oracle-backend host-delegated --attribution (scratch repo, trufflehog on PATH). Adding --scan-deep false saves all four values and plants the block in AGENTS.md only. On v0.9.0, whose docs target defaulted to both, the dropped docs target is what planted the block into CLAUDE.md as well as AGENTS.md (the default became skip in dae705d5).

## Evidence 2026-10-04 (a downstream lab)

A downstream private project's lab hit this on its first install: `--visibility`, `--docs-target` and `--oracle-backend` were not persisted. Reproduced at tip 57d5ec9fa, after the recent setup work (#793, #800, #805): `yes | abcd ahoy install --adopt --docs-target agents_md --visibility private --oracle-backend host-delegated --attribution` in a scratch repository with a stand-in trufflehog on PATH answers `scan_deep (true/false) [false]: y`, leaves `.abcd/config.json` holding only `attribution` and `meta`, and ends with remaining gaps `config.docs_target_missing`, `config.oracle_backend_missing` and `config.visibility_missing`, with no note that the scan_deep answer was refused. The lab rated it major.
