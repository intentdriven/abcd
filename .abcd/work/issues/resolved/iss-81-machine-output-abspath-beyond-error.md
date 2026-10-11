---
schema_version: 1
id: "iss-81"
slug: "machine-output-abspath-beyond-error"
severity: "minor"
impact: fix
category: "bug"
source: "user-observation"
found_during: "2026-07-12 iss-76 security re-review"
found_at: "internal/surface/cli/cli.go"
resolution: "Machine (--json) output rendered repo-relative everywhere: capture success/resolve/wontfix/list path fields and the memory ingest error path. Added fsutil.RepoRel canonical helper; detector in error_pathleak_surface_test.go covers success and error envelopes."
---

Machine output surfaces absolute local paths OUTSIDE the cli.Run error-identity scrub added in iss-76. Two instances found in the iss-76 security re-review: (1) 'memory ingest <abs-path>' — materialFromLocal EvalSymlinks-resolves the user's source argument and embeds the resolved absolute path in a custom IngestError (internal/core/memory/ingest.go:655,658,663,667). It lies outside cwd/home, so scrubPaths' root-redaction and PathError base-name reduction both miss it. Low severity: it is the user's own argument and carries no developer identity for out-of-home paths (a ~/x argument still redacts to ~/x). (2) STRONGER, separate surface: the capture SUCCESS envelope emits an absolute 'path' field, e.g. {"path":"<home>/.abcd/work/issues/open/iss-N-...md"} — a full home/cwd-rooted path emitted unbidden on the SUCCESS path, which the error-only scrubPaths never sees. Detector (per unrecognized-input-never-writes / no-absolute-paths-in-machine-output): extend the per-verb --json table to assert no absolute path in BOTH the success and error envelopes for verbs echoing store/source paths; fix by rendering repo-relative or base-name paths at those sites. Acceptance corpus: the ingest error sites and the capture success 'path' field above.