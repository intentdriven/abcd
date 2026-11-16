---
id: dcn-2611160234098258
slug: abcd-explain-json-writes-one-command-s
date: 2026-11-16
---
2026-11-16 — abcd explain --json writes one command's facts through EncodeCommandFacts, with no HTML escaping, so it is byte for byte that command's slice of its committed file; every other --json output keeps render's encoder; decided by the run for owner review

Grounds: Criterion 3 of itd-2610301346085246 asks that `abcd explain <path> --json` be byte-equal to that command's slice of its committed JSON file. The files are written by `EncodeCommandFacts` with `SetEscapeHTML(false)`, so a usage line's `<file>` reads as written. `render`, the encoder every other `--json` output shares, keeps Go's default escaping and prints `\u003cfile\u003e`. The drift test had re-encoded the file's slice with `render`, which compared explain with itself and hid the difference on every usage line or example holding `<`, `>`, or `&`; encoded with `EncodeCommandFacts`, the test failed on such commands. An agent reads explain's answer, so explain now writes its facts document through `EncodeCommandFacts`, and the duplicate `explainDocument` type gives way to `CommandFactsDocument`. Both encoders indent by two spaces and close with a newline, so the only bytes that change are the escapes. `explain --input <flag> --json` and every other `--json` output still go through `render`, unchanged; no test, snapshot or generated reference encoded explain's output.

Reverse it by encoding the per-command files with `render`'s settings in `EncodeCommandFacts` (internal/surface/cli/commandfacts.go), returning explain's facts answer to `render` (internal/surface/cli/explain.go), keeping the drift test's `want` on `EncodeCommandFacts`, and running `make command-facts-sync`.
