---
schema_version: 1
id: "iss-2611042320130590"
slug: "signal-tests-fail-under-an-ignored"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "a cloud session's preflight, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain_signal_surface_test.go"
remedy: "Have these tests start their child with SIGINT's default disposition restored, or skip with a named reason when it is ignored, and run them once under an ignored SIGINT to prove it."
---

The pty and drain signal tests fail when the test process was started from one that ignores SIGINT: a cloud session reported it on 2026-11-04. A child inherits the ignored disposition, which these tests do not set back.
