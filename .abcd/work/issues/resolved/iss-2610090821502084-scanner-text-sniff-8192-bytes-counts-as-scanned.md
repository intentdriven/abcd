---
schema_version: 1
id: "iss-2610090821502084"
slug: "scanner-text-sniff-8192-bytes-counts-as-scanned"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security report, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/scanner.go"
remedy: "On the text branch, require the whole file to be valid UTF-8 with no NUL or send it through `decodeContent`, anything not fully accounted being Unscanned; prove it with a scanner test (watched fail first) that the prose-plus-gzip file is refused, a normal markdown file still scans and the `.gz` control still hard-fails; sweep siblings (every other classification decided on a sniffed prefix)."
resolution: "The text branch now requires the whole file to be text (no NUL, valid UTF-8), not the first 8 KiB; a file that is not is Unscanned with its reason, so the launch gate refuses it."
impact: fix
---

The secret scanner counts a file as fully scanned when only its first 8192 bytes are valid UTF-8 text, so a gzip member after a page of prose in an included markdown file ships with no finding.

A private security report, fixed in this release; its advisory, with the full text and reproduction, is published with the release.

Evidence (lines at main 7549ca2d5): `isText` sniffs 8192 bytes (internal/adapter/scanner/scanner.go:1519). The text branch then runs `ScanText` on the raw bytes and increments FilesScanned without calling `decodeContent` (internal/adapter/scanner/scanner.go:1221-1225). `cover` already refuses an 8192-byte sniff as coverage of a decoded region. `docs/` is an include, and the bundler does not inspect content before inclusion.