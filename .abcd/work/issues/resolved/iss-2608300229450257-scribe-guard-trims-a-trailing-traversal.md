---
schema_version: 1
id: "iss-2608300229450257"
slug: "scribe-guard-trims-a-trailing-traversal"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "itd-188 fourth-round security review, 2026-08-30"
found_at: "internal/core/lint/scribecontract_test.go (trailing-punctuation trim)"
resolution: "The traversal check now runs on the raw match, before any trimming. Trailing sentence punctuation includes the full stop, so trimming first ate the second dot of a path whose final segment is .. — .abcd/work/issues/.. reached the prefix check as .abcd/work/issues, passed it, and admitted the whole of .abcd/work/. Three bypass cases pin it: bare, in inline code, and entity-encoded as &#46;&#46;, each watched red first."
impact: internal
---

The scribe guard's trailing-punctuation trim runs before the traversal and prefix checks and strips a full stop, so a path whose final segment is .. with no trailing slash (.abcd/work/issues/.., bare, in inline code, or entity-encoded) loses its traversal segment, passes both checks, and admits .abcd/work/ (DECISIONS.md, CONTEXT.md, reviews/, rulesets/) — exactly the case the root constant was narrowed for. Bounded to one level up; ../.. and ../ forms are still refused. Run the traversal check on the raw token before trimming, or refuse to trim a trailing full stop when the remainder would end in /. or /..
