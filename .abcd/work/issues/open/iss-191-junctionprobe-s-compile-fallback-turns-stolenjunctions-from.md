---
schema_version: 1
id: "iss-191"
slug: "junctionprobe-s-compile-fallback-turns-stolenjunctions-from"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bug-hunt loop round 3, pre-PR review of iss-188's fix"
found_at: "internal/adapter/scanner/scanner.go"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): When the fallback alternation cannot compile: cap the candidates, drop patterns until it compiles, or fail closed?"
---

junctionProbe's compile fallback turns stolenJunctions from a bounded candidate walk into a per-byte one, a latent cost cliff. junctionProbe (internal/adapter/scanner/scanner.go) builds one combined alternation over every pattern's boundary-free body; if that alternation fails to compile it falls back to regexp.MustCompile(`(?s).`), which matches at EVERY byte offset. The fallback was chosen so the candidate generator can over-produce but never under-produce — correct for detection, but it changes stolenJunctions' cost class: instead of one wholeMatch validation per real candidate junction in the backtrack window, wholeMatch runs once per byte offset in the window, and each of those validations is itself proportional to the match's own length (it re-runs the anchored probe over line[m.start:cut]). Cost per match goes from O(window) validations to O(window x match length) work. A reviewer measured roughly a 500x slowdown substituting the fallback directly — 4.21s against 8.25ms on one 200KB match. Not reachable with the current bundled pattern set: every bundled body alternates cleanly, and triggering the compile failure needs a pathological custom .abcd/config/pii.json override whose added pattern makes the joined alternation invalid (the per-pattern regexes are each compiled and validated on merge, so this is hard but not provably impossible to reach). Latent rather than urgent. Options if it is ever worth closing: cap the fallback's candidate count the way the window caps the search, drop patterns from the alternation one at a time until it compiles rather than abandoning the whole set, or mark the scanner unavailable (fail-closed) when the combined probe cannot be built — the last is consistent with how the package already treats a config fault.