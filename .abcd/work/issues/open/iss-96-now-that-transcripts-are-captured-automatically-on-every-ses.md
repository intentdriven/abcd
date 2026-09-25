---
schema_version: 1
id: "iss-96"
slug: "now-that-transcripts-are-captured-automatically-on-every-ses"
severity: "minor"
category: "security"
source: "manual-test"
found_during: "itd-89-m1"
found_at: "internal/adapter/scanner/patterns.go"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Transcript-path secret detection: an entropy detector, key-name context, or an opt-in external scanner?"
---

Now that transcripts are captured automatically on every session end, the scanner's secret-pattern coverage becomes load-bearing in a way it was not when capture was a manual verb nobody ran. Verified by live test: the bundled patterns DO catch anchored tokens (AKIA... access key IDs, ghp_/gho_/sk-ant- style prefixes) and absolute home paths, but they do NOT catch unanchored high-entropy values — an AWS SECRET access key (the 40-char base64 value, no prefix), a bare password, or a generic API token with no recognisable prefix all pass through into the store verbatim. This is the standard prefix-matching limitation and is pre-existing, not a regression; the point is that the blast radius changed. Consider entropy-based detection or the opt-in gitleaks adapter for the transcript path specifically, where the input is unstructured prose rather than curated source.

---

**Verification (2026-08-01, item A6 of the v0.5.0 plan).** Re-checked against the
network-identifier pattern set landed by A2, which is folded into
`DefaultPatterns` and therefore inherited by the transcript path. Outcome: one
class newly covered, the three classes this entry names confirmed as residue by
test rather than by inspection. The entry stays open, re-scoped below.

**Newly covered since capture.** Nothing in the entropy classes, but the
transcript path gained a detection class it did not have when this was filed:
non-reserved IPv4/IPv6/MAC addresses (`net:ipv4`, `net:ipv6`, `net:mac`,
hard_fail) and LAN/device hostnames (`net:lan_hostname`, `net:device_hostname`,
warn), by allowlist inversion over the reserved documentation ranges. All five
kinds are exercised on the transcript path itself in the corpus below, not
inferred from their presence in `DefaultPatterns`. A leaked machine identifier is
therefore newly covered on this path since this entry was captured. It was never
part of the residue — line 12 above names unanchored secrets and home paths only,
never a network identifier — so this is coverage the re-check establishes, not a
gap it closes.

**Residue, verified empirically.** Synthetic specimens presented to the
transcript path's own entry point — `(*Scanner).ScanText`, the call
`internal/core/history.Capture` makes at stage one — produce zero findings:

- bare secret-access-key value (40 characters, no prefix), both with and without
  an `AWS_SECRET_ACCESS_KEY=` key name in front of it — passes through;
- bare password, both after `password:` and after `--password=` — passes through;
- prefix-less API token (32 hex characters after `api_key =`; 40 base64
  characters after `Authorization: Bearer`) — passes through;
- a genuinely high-entropy 40-character alphanumeric value (5.32 bits per
  character, assembled at run time from a fixed-seed shuffle), both bare and
  after an `api_key =` key name — passes through.

Anchored controls in the same corpus are caught (`token:aws_access_key`,
`token:github_pat`, `token:anthropic`, `token:pem_private_key`,
`home_path_self`, `home_path_other`, plus all five network kinds above), so the
specimens reach a working detector; one of those controls sits inside the
negative test too, so an emptied pattern set cannot satisfy it vacuously. The
TOKEN patterns are prefix-anchored apart from `rp_session_key`, which keys on one
literal JSON field name (`"sessionKey"`) rather than on a generic key-name class,
and no pattern measures entropy — so a `password:`, `api_key =` or
`Authorization: Bearer` key name beside a value does not help. The set as a whole
is not uniformly prefix-anchored, and the entry should not be read as saying so:
the network kinds are an allowlist inversion over the reserved documentation
ranges, the identity kinds key on the probed identity, `token:pem_private_key` is
a literal header, and `Pattern.SkipAt` (in
`internal/adapter/scanner/patterns.go`) already lets a pattern accept or
reject a match by what SURROUNDS it on the line. None of those mechanisms reaches
an unlabelled value, which is why the residue stands.

**Pinned in the tree**, so the residue is evidence rather than a claim:
`TestTranscriptPathMissesUnanchoredEntropy`
(`internal/adapter/scanner/transcript_coverage_test.go`) at the pattern-set
boundary, and `TestCaptureStoresUnanchoredEntropyVerbatim`
(`internal/core/history/history_test.go`) at the store boundary, which captures a
six-line transcript carrying an anchored token on the first line and the
unanchored specimens on the lines below it, then asserts each specimen present
verbatim in the written record while the anchored token is absent AND its
`fingerprintSpan` fingerprint present — that is the byte-level mirror of
`maskSecret`, and it is what the Capture path actually produces, via `sealLine`;
absence alone would also be satisfied by a
store that dropped the line. The same capture asserts the contra case, so the
class A2 did close is evidence rather than prose: a flagged private address on
the last line is absent from the stored record and its `[redacted-address]`
placeholder present.

**What the pins can and cannot catch.** They assert CURRENT behaviour, and their
reach is the PATTERN SET this path runs. They alarm reliably for growth in that
set — a charset/length floor, key-name context matching, any new `DefaultPatterns`
entry THAT REACHES THESE SHAPES, and now a genuine entropy floor. (The
qualification matters and the test file states it: an entry that reaches nothing
in the corpus leaves both pins green, and rightly so.) That last one needed fixing: the original
specimens all repeat, and measure 3.12 (`FAKEfake00`×4), 2.16 (`deadbeef`×4) and
2.75 (base64 run) bits per character, all BELOW the ~3.5-bit threshold an entropy
detector conventionally uses, so such a detector could have landed with every pin
still green. A high-entropy specimen was therefore added at both boundaries —
5.32 bits per character, assembled at run time from a fixed-seed shuffle of the
alphanumeric alphabet, with its measured entropy asserted in-test (at or above
4.5 bits per character) so it cannot quietly degrade into a repetitive value, and
its exact value asserted against a golden constant on BOTH sides, so the two
duplicated generators cannot silently diverge and determinism is pinned as well.
They CANNOT alarm for option (c) below: an opt-in external-scanner adapter never
consults `DefaultPatterns`, so `ScanText` keeps returning nothing and both pins
stay green while coverage has in fact grown. That route must be re-pointed by
hand, and the pins alone are therefore not grounds to close this entry.

**Re-scoped to a decision, not an implementation.** What remains is a choice
between three options that differ in REACH, not only in false-positive cost.
Stating them as equal-coverage-at-different-cost would misdescribe them:

- **(a) an entropy/charset-class detector with a length floor.** The only one of
  the three that reads the VALUE, so the only one that reaches an unlabelled
  value: it would cover every specimen above, including the bare 40-character
  secret-key shape with no key name beside it.
- **(b) key-name context matching** (`password`, `secret`, `token`, `api_key` to
  the left of a delimiter). Labelled values ONLY. It is structurally incapable of
  this entry's own first-named case — a bare secret-access-key value with no key
  name, pinned as the case "bare 40-char secret-key shape, no key name" — because
  there is no key for it to match. Its merit is elsewhere: it generalises
  mechanisms the set already ships rather than introducing one, since
  `rp_session_key` matches on a literal field name today and `Pattern.SkipAt`
  already decides a match by its surrounding line. Note against the measurement
  below: (c) as shipped already reaches most of what (b) would, so (b)'s distinct
  increment over (c) is narrow — labelled values that (c)'s delimiter or entropy
  conditions exclude — and its real argument is that it is native, in-tree and
  carries no external dependency, not that it reaches further.
- **(c) the opt-in external-scanner adapter over the transcript path only.**
  Reach is whatever the adapter's ruleset delivers — a measurement, not an
  assumption. Measured below: MORE than (b), not less.

**The (c) measurement, re-run 2026-08-01.** An earlier measurement recorded on
this entry was wrong in both its fixture and its conclusion, and is superseded by
this one. Method, so it can be repeated: gitleaks **8.24.3**, the release binary
`gitleaks_8.24.3_linux_x64.tar.gz` verified against the release `checksums.txt`
(sha256 `9991e0b2903da4c8f6122b5c3186448b927a5da4deef1fe45271c3793f4ee29c`) — the
same version this repo's own history scan pins in `.github/workflows/ci.yml` —
run as `gitleaks dir` with its DEFAULT rules, no repo config. The fixture is one
specimen per line, and it is named precisely: every case of the table in
`TestTranscriptPathMissesUnanchoredEntropy`
(`internal/adapter/scanner/transcript_coverage_test.go`), every line of the
transcript `TestCaptureStoresUnanchoredEntropyVerbatim` captures
(`internal/core/history/history_test.go`), and delimiter probes on the same
values, added to isolate the mechanism. Twenty-one lines, six findings, all
`generic-api-key`. Enumerated in full below — sixteen rows for twenty-one lines,
because a shape carried by both fixtures is merged into one row (differing only
by the store transcript's conversational framing: its `user: `/`assistant: `
prefixes, and on row 1 its prose lead-in around the same anchored token), and
rows 10 and 13 merge the two undelimited key names. (`<hi>` is the 40-character
high-entropy specimen, `<pass>` the five-word passphrase; neither value appears
here. The entropy column is gitleaks' reported measurement on findings and the
same Shannon computation on passes — identical arithmetic, since gitleaks
reports no figure for a line it does not flag.)

| # | line as scanned | from | gitleaks 8.24.3 |
|---|---|---|---|
| 1 | `token=ghp_` + `F`×36 | both fixtures | passed (see caveat) |
| 2 | `AWS_SECRET_ACCESS_KEY=` + `FAKEfake00`×4 | both fixtures | passed — entropy 3.12 |
| 3 | `FAKEfake00`×4, bare | scanner corpus | passed — no key name |
| 4 | `password: ` + `<pass>` | both fixtures | **CAUGHT** `generic-api-key`, entropy 3.66 |
| 5 | `--password=` + `<pass>` | scanner corpus | **CAUGHT** `generic-api-key`, entropy 3.66 |
| 6 | `api_key = ` + `deadbeef`×4 | both fixtures | passed — entropy 2.16 |
| 7 | `Authorization: Bearer ` + `Zm9vYmFy`×5 | scanner corpus | passed — entropy 2.75 |
| 8 | `<hi>`, bare | scanner corpus | passed — no key name |
| 9 | `api_key = ` + `<hi>` | scanner corpus | **CAUGHT** `generic-api-key`, entropy 5.32 |
| 10 | `session_token ` + `<hi>` (no delimiter) | store fixture | passed — no `=`/`:` |
| 11 | `session_token = ` + `<hi>` | delimiter probe | **CAUGHT** `generic-api-key`, entropy 5.32 |
| 12 | `session_token: ` + `<hi>` | delimiter probe | **CAUGHT** `generic-api-key`, entropy 5.32 |
| 13 | `session_token ` / `api_key ` + `<hi>` (no delimiter) | delimiter probe | passed — no `=`/`:` |
| 14 | `session_token = ` + `FAKEfake00`×4 | delimiter probe | passed — entropy 3.12 |
| 15 | `api_key = ` + `Zm9vYmFy`×5 | delimiter probe | passed — entropy 2.75 |
| 16 | `ssh ` + private IPv4 quad | store fixture | passed — not a credential rule's business |

The rule doing all the work is `generic-api-key`, and its condition is a
conjunction of three things — a key-name KEYWORD (`key`, `token`, `secret`,
`password`, `auth`, `api`, …), a DELIMITER (`=`, `:`, and relatives) between
keyword and value, and Shannon entropy of at least **3.5 bits per character** in
the value — subject to the ruleset's own stopword allowlist, which suppresses a
value containing a dictionary marker such as `fake` or `deadbeef` regardless of
its entropy. Rows 4, 5, 9, 11 and 12 satisfy all three. Every other row fails at
least one: rows 3 and 8 have no keyword; rows 10 and 13 have a keyword but no
delimiter; rows 7 and 15 have keyword and delimiter but sit under the entropy
floor; rows 2, 6 and 14 sit under the floor AND carry the `fake`/`deadbeef`
stopwords, either of which suppresses them alone — raising their entropy without
changing their spelling would not flip them.

**(c)'s reach, stated accurately.** Keyword + delimiter + entropy ≥ 3.5 reaches
LABELLED, DELIMITED, high-entropy values — including the branch's own
`api_key = <hi>` corpus case. It therefore SUBSUMES much of option (b) and adds
an entropy floor on top of it, rather than being the narrowest of the three. It
does NOT reach a bare value with no key name (rows 3, 8 — this entry's own
first-named case), a key name separated from its value by prose rather than a
delimiter (rows 10, 13 — exactly the store fixture's hand-built
`session_token <hi>` line), or any labelled value below the entropy floor (rows
2, 6, 7, 14, 15 — every repetitive specimen in the corpus, whatever key name
precedes it).

**Caveat on row 1**, verified rather than assumed: the anchored `ghp_` control
passed because its payload is one character repeated 36 times, which gitleaks'
own entropy filter drops. A realistic-entropy `ghp_` token in the same run IS
caught, by the `github-pat` rule. Row 1 is a fact about this fixture, not about
that ruleset's coverage of real prefixed tokens.

**What the earlier measurement got wrong**, recorded so the correction is not
mistaken for a change in the tooling. It named "the store fixture's own specimen
set" and then listed a base64 token among the passes — but the store fixture
carries no base64 token (`Zm9vYmFy`×5 lives only in the scanner corpus), so the
fixture it described did not exist. And it read the high-entropy miss as a
failure of key-name context ("despite a `session_token` key name beside it"),
which inverted the causal claim: the miss is the absent DELIMITER on that
hand-built prose line, as rows 10 to 13 isolate. The consequence is a materially
different answer — (c) reaches labelled high-entropy values, so it is not the
narrow option the earlier passage implied.

False-positive cost is the second axis, and it lands hardest on (a): a transcript
is full of hashes, base64 blobs and identifiers that are not credentials, and a
redaction false positive corrupts the record redaction exists to preserve. (c)
carries a share of that cost too, in proportion to its reach — its entropy floor
fires on any labelled high-entropy token in prose, credential or not. Reach and
cost together are the open question, and where the bar sits is the maintainer's
to decide — grill-then-implement, not autonomous work.
