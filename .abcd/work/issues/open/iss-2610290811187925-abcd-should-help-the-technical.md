---
schema_version: 1
id: "iss-2610290811187925"
slug: "abcd-should-help-the-technical"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "coordinator session, product-thinker ruling on personal identifiers, 2026-10-29"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/20-banlist.md"
remedy: "Add an abcd step (through ahoy and the GitHub settings audit of itd-2610290746077716) that: derives the private-names layer deterministically on every run from the machine's host and computer names, the local user accounts and the forge's org members and collaborators, never printing them; scans the tracked tree and abcd's own pushed refs for those names and reports counts only; proposes the replacements (roles, the organisation, a team handle, persona names in fixtures) for a yes; and for the web-only steps (creating a visible team with write access) gives exact prefilled links, then verifies the result through the API <!-- record-lint: forward-looking -->"
---

abcd should help the technical facilitator of every abcd-managed repository set up the same personal-identifier protection this repository now has, as easily as possible. That means: no org member's login or name in any tracked file or anything abcd pushes or publishes (records, fixtures, credits, the site's contributors page, reservation and announcement trailers and ref names), with commit author fields left as the person's identity; code-owner sign-off through an org team rather than named people; test fixtures that use the standard abcd personas from .abcd/development/personas.json instead of invented or real names; and a private-names list that always and automatically includes the machine's names, its local user accounts and the org's members and collaborators. Today each of these was set up by hand in one repository, and a managed repository gets none of it.
