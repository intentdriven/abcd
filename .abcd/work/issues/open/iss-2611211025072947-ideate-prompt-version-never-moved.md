---
schema_version: 1
id: "iss-2611211025072947"
slug: "ideate-prompt-version-never-moved"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "a /abcd:ideate run in the abcd source checkout, 2026-11-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ideate/record.go"
remedy: "Raise the page prompt_version with every change to what a leg asks for, have ideate record hold the version its validator expects, write the version into the verdict it records, and refuse an older one with a message that says the rules for the reviewers changed and names both versions and the page to read, and add a check that fails when the protocol sections of the page change while the version stands still, as the agent contract check does for agent prompts (iss-2610111600065621). An older page's payload is then refused on its version before its missing field is named, which dcn-2610311344529366 declined for the same change, so the change records a decision that supersedes that one, or takes that decision's own way back (schema_version 3) where that is the smaller change. Grounds: internal, the prompt version stamp of itd-5."
---

The `prompt_version` of an ideate verdict did not move when the protocol changed. The page at v0.16.0, at v0.17.0 and at the tip all give `"prompt_version": "1.0.0"`, though between the first two the page gained the current-state mode, which changes what every leg 3 reviewer is told (iss-2610310815034547). `ideate record` checks only that the field is a semantic version (internal/core/ideate/record.go:526) and then drops it: the written verdict states no version. The recorder does refuse an older `schema_version` and names the version to compose, but that stood at 2 across the same change, so that gate could not fire either. That it stood at 2 was decided at the time (dcn-2610311344529366): a payload without the field is refused by a message that names the field and both of its values, so the host learns what to add, and a refusal that names a version was judged to add nothing. The runs of 2026-11-20 are the case that ground did not cover: the host learns what to add to the document, not what to tell the reviewers. So the recorder cannot tell a verdict composed from an older page from one composed from its own, and nothing in the document says which version of the rules the reviewers were given. A host that meets the refusal for a missing `current_state` can add that one field and record a verdict that states ignore mode over reviews never given its rule. Two runs met that refusal on 2026-11-20 and ran or resumed their reviewers again by their own choice; nothing required it. The remedy does not close that part: a host can raise the version by hand as it can add the field, so the version, once the verdict records it, says which rules a run claims to have followed and proves nothing about what its reviewers were given.
