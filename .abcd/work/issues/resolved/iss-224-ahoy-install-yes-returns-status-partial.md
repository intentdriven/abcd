---
schema_version: 1
id: "iss-224"
slug: "ahoy-install-yes-returns-status-partial"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "ahoy install dogfood"
found_at: "internal/core"
resolution: "Observed on the month-stale pre-iss-171 repo-root binary; the /usr/local/bin refusal scenario no longer exists after rebuild. The surviving current-code concern (silent bare-return failure paths in installDevShim) is recaptured precisely as a separate issue."
impact: internal
---

ahoy install --yes returns status=partial with remaining symlink.missing but emits no note explaining the refusal (unwritable /usr/local/bin); declined_categories is null and notes are absent, so the user cannot tell why the symlink was skipped or what to do next