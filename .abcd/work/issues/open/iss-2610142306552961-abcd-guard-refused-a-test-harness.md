---
schema_version: 1
id: "iss-2610142306552961"
slug: "abcd-guard-refused-a-test-harness"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "brief session page security round, live harness run"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/session/web/session.js"
remedy: "Read a named script for shell hazards only when it is a shell script (by shebang or extension), or parse JavaScript files as non-shell; add a test with the varint line"
---

abcd guard refused a test-harness command as killall-by-name because a JavaScript file the command named contains the line 'num += (b & 0x7f) * mult;', a bitwise AND on a varint byte, which the script reader took for a shell kill
