---
schema_version: 1
id: "iss-2610151411351873"
slug: "internal-ghcontract-checks-that-every"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "abcd land review round 5, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/ghcontract/graphql.go"
remedy: "Pin each GraphQL field's arguments (name and type, read by a one-off introspection query) beside the field pins in internal/ghcontract, and have CheckGraphQL refuse a query whose field omits a required argument or passes an unknown one, so a dropped isRequired(pullRequestNumber:$n) fails in the fake forge as it does on the real one"
resolution: "internal/ghcontract pins every pinned GraphQL field's arguments (graphql-args.txt, read by schema introspection) and CheckGraphQL refuses an unknown argument, a required one left out, and isRequired with neither pullRequestId nor pullRequestNumber (a 'one' pin from the forge's run-time refusal, since its schema marks neither non-null)"
impact: internal
resolved_by:
  commit: "90e5163ebaed24b3135bed7dcb12a91daa6483e8"
---

internal/ghcontract checks that every field a GraphQL query selects exists on its type, but never checks the field's arguments. abcd land's queue-state query depends on isRequired(pullRequestNumber:$n): the forge refuses isRequired with no argument ('A pull request ID or pull request number is required'), yet a query that dropped the argument would pass every test against the fake forge. That is the class of defect the gh field pins exist to stop (a field the real forge refuses, hidden by a fake). Found by the round 5 implementer of the abcd land change, 2026-10-15.

## Grounds

- pursued: a query that drops isRequired(pullRequestNumber:$n), Repository.pullRequest(number:) or passes an argument a field lacks now fails CheckGraphQL in the fake forge; it would be shown wrong by a query the real forge accepts being refused (every abcd query checked by the landing, landings, autonomy, claim and loop tests still passes) or by a dropped required argument still passing
