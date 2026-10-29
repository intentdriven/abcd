---
schema_version: 1
id: "iss-2610262036082275"
slug: "itd-134-renders-a-banner-from-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 626 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-134-every-abcd-managed-cli-greets-like-abcd.md:13"
remedy: "In itd-134 (title and why-it-matters) say the banner renders from the identity block's title and tagline (and pitch if wanted), and name where the version comes from (the repo's own release or build metadata), since the block has no version field."
related_intents: [itd-134]
finding_ref: cfk-d2d16231fd872e8f
---

itd-134 renders a banner from the identity block's name, tagline and version; 19-identity's block holds a title, a tagline and an optional pitch, and no version.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-134-every-abcd-managed-cli-greets-like-abcd.md:13` — "rendered from that repo's canonical identity block — name, tagline, version"
- End B: `.abcd/development/brief/04-surfaces/19-identity.md:50` — "Title and tagline are required; the pitch is optional at onboarding and may wrap"

itd-134 scaffolds a banner rendered from the repo's canonical identity block, naming its fields as name, tagline and version. 19-identity defines the block's fields as Title, Tagline and Pitch, with title and tagline required and the pitch optional. The block carries no version and calls the name a title, so a banner sourced from the block as itd-134 describes cannot take its version from it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 626.
