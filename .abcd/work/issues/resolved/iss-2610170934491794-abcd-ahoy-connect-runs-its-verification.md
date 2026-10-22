---
schema_version: 1
id: "iss-2610170934491794"
slug: "abcd-ahoy-connect-runs-its-verification"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-166 build, 2026-10-17"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: "internal/core/oracle/config.go"
remedy: "Have ahoy connect retry its one verification call without the usage request when the refusal names it, and write stream_usage: false into the provider block it creates, saying so on the setup screen; the call stays inside the setup the person started."
resolution: "ahoy connect asks its verification call once more without stream_options.include_usage when the first is refused with HTTP 400 or 422, writes stream_usage false into the block it creates and says so in its text and JSON output; --stream-usage sets the value outright"
impact: fix
resolved_by:
  intent: "itd-166"
---

abcd ahoy connect runs its verification call before any provider block exists, so a server that refuses the usage-in-stream request (stream_options.include_usage, added by itd-166) fails setup, and the refusal names the stream_usage setting on a block the person can only add by hand.
