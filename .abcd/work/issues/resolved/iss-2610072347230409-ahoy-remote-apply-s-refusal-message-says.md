---
schema_version: 1
id: "iss-2610072347230409"
slug: "ahoy-remote-apply-s-refusal-message-says"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "review of the ahoy exit-code lane (run-2610072140293887), 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/remote.go"
remedy: "Make ahoy remote apply's refusal say which toggles already landed on the forge when it refuses after one (the remote.go paths that reach refused after a toggle), or refuse before the first toggle; never print 'nothing was changed' after a change landed."
resolution: "ahoy remote apply now names the toggles it already changed on GitHub when it refuses after one, and no longer says nothing was changed."
impact: fix
resolved_by:
  commit: "13be4fb0f244a40b2570816414509cecceaa8ecd"
---

ahoy remote apply's refusal message says nothing was changed, but two paths in internal/core/ahoy/remote.go reach the refused status after a toggle has already landed on GitHub, so the person is told nothing changed when one setting did.
