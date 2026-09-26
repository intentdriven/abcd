---
schema_version: 1
id: "iss-2609260221578565"
slug: "barerender-s-exception-list-gives-statusline-the-reason-with"
severity: "nitpick"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/barerender.go"
---

barerender's exception list gives statusline the reason 'with no payload there is no row to render', which is false: bare abcd statusline with stdin at /dev/null renders a row and exits 0, so it is not an exception to the bare-render property.
