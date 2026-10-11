---
schema_version: 1
id: "iss-107"
slug: "ahoy-install-visibility-v-yes-reports"
severity: "minor"
impact: fix
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "apply-as-update: an explicit --visibility/--docs-target/--oracle-backend/--scan-deep override now overwrites an already-valid persisted value and echoes the change; shared applyOverride path covers all four slots; Install no longer short-circuits when an override differs"
---

ahoy install --visibility <v> --yes reports 'already up to date' without applying the explicitly requested visibility on an already-configured repo: skip-if-set beats an explicit flag — stepConfigValues short-circuits when repo.visibility is validly set (internal/core/ahoy/apply.go:196) and ValueOverrides are only consulted on the config-gap path (apply.go:179->220), so the flag silently no-ops; an explicit flag value should either apply as an update or error loudly, never silently skip (maintainer hit this from another repo and hand-edited its config)