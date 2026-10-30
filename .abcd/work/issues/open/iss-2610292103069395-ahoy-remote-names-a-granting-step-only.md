---
schema_version: 1
id: "iss-2610292103069395"
slug: "ahoy-remote-names-a-granting-step-only"
severity: "minor"
category: "drift"
source: "impl-review"
found_during: "fidelity review rcp-6083de2cef52 of itd-2610290746077716"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/remote_baseline.go"
remedy: "Have every 403 or 404 a remote read meets say what the sign-in lacks and how to get it (the admin role on the repository, or the gh scope and its refresh command), and have a failed plan read say that the plan could not be read and why, with a test per read"
---

abcd ahoy --remote names the missing scope and the command that grants it only for the organization's Actions policy. The fidelity review of itd-2610290746077716 (rcp-6083de2cef52, ac-8 met with concerns) found that a 403 or 404 on a repository settings read says the sign-in needs admin rights on the repository but gives no command or next step (internal/core/ahoy/remote_baseline.go:459). A failed read of the owner's plan returns an empty plan and no reason at all (remote_baseline.go:478), so the person cannot tell why the plan-dependent settings read as they do.
