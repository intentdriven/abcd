---
id: dcn-2610180720415626
slug: refusing-a-newer-coordination-record
date: 2026-10-18
---
2026-10-18 — Refusing a newer coordination record starts with the release that adds a version (facilitator-on-delegation, itd-2610170634394993 planning prep): readers already shipped ignore an unknown trailer, so the criterion's refusal can hold only from the release that adds a version field, and the criterion says so (grounds: internal/core/claim/message.go; kubernetes.io version-skew-policy)
