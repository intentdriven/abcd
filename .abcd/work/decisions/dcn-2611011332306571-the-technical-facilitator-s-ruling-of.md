---
id: dcn-2611011332306571
slug: the-technical-facilitator-s-ruling-of
date: 2026-11-01
---
2026-11-01 — The technical facilitator's ruling of 2026-11-01 on publishing schemas: abcd generates JSON Schemas for its boundary documents from the Go types in this repository, record front-matter included and run-state excluded, gated against drift and shipped with each release under a stable location-independent $id namespace; the work folds into itd-113, whose MCP front door needs per-tool input schemas anyway, and the OpenAI-compatible oracle adapter sends a reply's schema with each request so a local model server can constrain its answer. A separate schema repository, superseding adr-28, is reopened only when a non-Go implementation needs its own cadence or a second organisation proposes a schema change. Reverse by superseding adr-28.
