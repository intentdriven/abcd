---
schema_version: 1
id: "iss-2608301649337920"
slug: "the-unsafe-finding-on-an-admissions-path-tells-the-operator"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-189-round-5-security"
found_at: "internal/core/lint/readingoutstanding.go"
---

the unsafe finding on an admissions path tells the operator abcd capture refuses it too when capture has no admissions code at all

Found by the round-5 security review. Branch-introduced by routing, not by
authorship.

Every `Unsafe` finding on an admissions path renders:

```
supports no claim either way. `abcd capture` refuses the same paths outright,
because its read is followed by a write
```

`core/capture` has no admissions code at all. `AdmissionsDir` and
`SurprisesDir` appear only in `core/lint` and `core/issueschema`, so there is no
read, no write, and no refusal. The operator is sent to look for a second gate's
agreement that nobody performs.

The sentence is PRE-EXISTING and was TRUE for the readings and dispositions
paths it was written for. Routing the admissions paths into it is what this
branch did, which is why the record is branch-introduced even though the words
are not.

It is also the exact mistake `readerFailsClosed` was invented to prevent, in the
same round that invented it: a message telling its reader that some other
component refuses the thing, when that component does not know the thing exists.
The record_schema legs were given a declaration; this one, in the report's own
file, was missed.

Remedy: drop the clause on the admissions path, or gate it the way
record_schema's own legs are gated — `readerFailsClosed` for the two that read a
record's properties, `readerRefusesDuplicateKey` for the duplicate, which is a
separate reader question (iss-2608301656200729).
