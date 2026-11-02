---
id: dcn-2610311344529366
slug: the-ideate-verdict-payload-gains-current
date: 2026-10-31
---
2026-10-31 — The ideate verdict payload gains current_state as a required top-level field while staying schema_version 2 (iss-2610310815034547). Grounds: the version number exists to stop a host composing against an old page from being misread in silence, and a payload without the field cannot be misread: it is refused with a message naming current_state and both modes, so the host that sent it learns exactly what to add. A version bump would refuse every old payload with a message about the version rather than the field, and would churn every fixture for no extra protection. The field sits at the top level rather than on the adversarial leg because it is the person's choice for the run, stated before the legs start. Reverse by bumping SchemaVersion to 3 in internal/core/ideate and teaching olderSchemaError that version 3 adds current_state.
