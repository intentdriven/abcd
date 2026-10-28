---
schema_version: 1
id: "iss-2610121448011094"
slug: "the-dogfood-remote-check-remoterefusal"
severity: "nitpick"
category: "bug"
source: "impl-review"
found_during: "security review round 3 of fix/hooks-run-head-in-dogfood-roots, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/dogfood/dogfood.go"
remedy: "test for a leading '[' before the '::' helper check and refuse a bracketed scp-like host with a reason that names it"
resolution: "a bracketed scp-like host is refused with its own reason"
impact: fix
resolved_by:
  commit: "75ba332ede4b19d37c310ad5e419062e0e936749"
---

The dogfood remote check (remoteRefusal, internal/core/dogfood/dogfood.go) refuses an scp-like IPv6 remote such as [::1]:path or [2001:db8::1]:path with the remote-helper reason, because the text contains '::'. The refusal is right but the reason is wrong: git does not read a '['-led remote as a helper.
