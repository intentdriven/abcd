---
schema_version: 1
id: "iss-2608301519255871"
slug: "the-proposal-join-resolves-handles-numerically-while-the-rep"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-3-security"
found_at: "internal/core/lint/schema.go"
resolution: "checkRecordJoins judges the SPELLING of a join that declares a family, in two places. What the value alone settles - the family's own prefix, lower case, nothing around it - is judged before anything is resolved, and takes five of the six spellings plus a cross-family handle. Whether the PADDING is the padding that admits is a fact about the target's FILE, since the family's reader keys on the filename and never opens the target's id property, so it is judged against the resolved target's filename and takes the sixth. A target whose filename is not itself a bare handle is one that reader does not read at all and stays silent (iss-2608300929274006). occasioned_by, which declares no family, keeps its prose tolerance."
impact: fix
resolved_by:
  intent: "itd-189"
---

the proposal join resolves handles numerically while the report keys them as strings so six spellings pass the gate green and admit nothing

Found by the round-3 security review. Pre-dates round 3 on the branch (the adm
store's join landed in round 2); does not reproduce on main.

`checkRecordJoins` resolves the target NUMERICALLY (`anyHandleFullRe` +
`strconv.Atoi`, schema.go:831-842) while `admittedProposals` keys it as a raw
STRING (`issueScalar`, readingoutstanding.go:476,485). So six spellings pass the
gate with ZERO findings and admit nothing, permanently and silently, probe-
verified: `"RDI-2608300000000002"`, `"Rdi-…"`, `"rdi-02608300000000002"`,
`"rdi-… "` (trailing space inside the quotes), `" rdi-…"`, and bare prose. The
first three RESOLVE and match the bucket, so the gate actively approves them;
the rest fall through the `m == nil → continue` "prose is legitimate" branch.

Meanwhile `reading_outstanding` goes on printing that the proposal has neither
an admission nor a decline while the admission record sits in the tree. That is
verbatim the harm iss-2608301327013320 (major) was opened and closed for: round
3 closed the CROSS-RUN spelling and left six.

The contrast is one leg away: the sibling `run` field is compared as a literal
string (`isAbsentValue(got) || got == r.bucket`, schema.go:911) and catches
every spelling tried against it.

Remedy, one place: a join that declares `sameBucketAs` is a join whose value the
schema says must be a handle of that family, so for such a join a value that is
not verbatim `^<family>-[0-9]+$` — no case shift, no leading zero, no padding
surviving the quotes — is a finding rather than a `continue`. `occasioned_by`,
which declares no family, keeps its prose tolerance untouched.
