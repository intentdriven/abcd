---
schema_version: 1
id: "iss-2611170055409898"
slug: "two-refusals-still-quote-text-a"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "reviews of the iss-2611170005148886 fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/identity/migrate.go"
remedy: "Name an unknown pin member only when its name is a short plain identifier and otherwise count it, and have the restamp refusal say the record's origin is outside the vocabulary without the value; a test with a prose key and a prose origin shows each stays out."
resolution: "The legacy pin's blocker names an unknown member only when its name is a plain identifier of at most 40 characters and counts the rest, and capture's restamp refusal says the record's origin is outside the vocabulary and names the vocabulary through provenance.OriginList, without the value. TestMigration_AProseMemberNameIsCountedNotQuoted, TestUnknownMembersNamesPlainOnesAndCountsTheRest and TestRestampRefusesAnOriginOutsideTheVocabulary hold it."
impact: fix
---

Two refusals still quote text a repository file chose: identity.PlanMigration joins a legacy pin's unknown member names word for word into the blocker the install's detection shows the person and the agent, unquoted, and capture's restamp refusal quotes a ledger record's out-of-vocabulary origin twice (its own %q and ParseOrigin's).

## Grounds

- pursued: neither refusal carries text a repository file chose beyond a plain member name, while each still says what to correct; a refusal the person cannot act on would show it wrong
