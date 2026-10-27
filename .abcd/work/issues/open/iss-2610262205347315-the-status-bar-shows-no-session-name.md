---
schema_version: 1
id: "iss-2610262205347315"
slug: "the-status-bar-shows-no-session-name"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run 2026-10-26, product thinker request"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusline/payload.go"
remedy: "Find from the harness's current status-line documentation which field, if any, carries the session's name, and read that; when the payload carries none, fall back to the persona name abcd's session-start hook reserved for this session, looked up by the payload's session id in abcd's own name registry; add a test fed from a captured real payload so the field name cannot drift silently."
---

The status bar shows no session name (Alice, Bob, ...) although itd-2610111707085098 shipped it in v0.15.1: the product thinker asked on 2026-10-26 for the name to be on the bar. The renderer works: the installed v0.15.1 'abcd statusline', fed a payload carrying session_name 'Alice', renders 'abcd-managed · Alice · abcd · main · ...', and without the field it renders no name. So the live bar lacks the name because the payload the harness hands the status line carries no session_name for these sessions (the field abcd reads may not be the one the harness sends), and sessions started before v0.15.1 never took a persona name at all (they list under names like abcd-0d, while sessions started since, such as Frank and Grace, carry persona names).
