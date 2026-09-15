---
schema_version: 1
id: "iss-2609090932441377"
slug: "the-raised-pem-opener-bar-drops-a-truncated-key-body"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/pem.go"
resolution: "The single-line opener no longer decides on width. A narrow run of sixteen to thirty-nine characters opens a block again, but only on two grounds that width says nothing about and that both have to hold: the header was PASTED, meaning its marker is the whole content of its line rather than named inside a sentence that goes on past it, and the run carries a byte of the base64 alphabet outside the letters — a digit, a plus, a slash, or the pad. The first refuses the mention that the width bar was raised to close; the second refuses an identifier under a header that was pasted. Neither reads entropy, deliberately: entropy would refuse this rule's own padding-tail case, and where an entropy bar sits is the open decision iss-96 holds for the maintainer. The armour, characteristic-width and consecutive-pair routes are unchanged. Cost, stated in both directions: a real body of that width is still missed where its header carries prose after the marker or where its run happens to be letters only, which is about three sixteen-byte runs in a hundred and one thirty-two-byte run in a thousand. The stage-two rescan is confirmed to have no independent reach here and could not have caught this: it filters stage-one findings rather than detecting, and no bundled pattern sees a headerless base64 line, which is iss-96's residue and already stated in pem.go."
impact: fix
---

Raising the single-line PEM opener bar from sixteen characters to forty removed coverage that the previous bar had. An open block, one carrying no END marker, whose visible body is a single line between sixteen and thirty-nine characters followed by any non-body line is no longer redacted, where it was before. The reachable shape is a real one rather than a contrived one: a private key re-wrapped to a narrow column by a mail client or a terminal, or truncated by a log rotation, leaves a header and one short line, and MIIEowIBAAKCAQEA, the canonical opening bytes of a DER-encoded RSA key, then survives verbatim. The failure is silent in the worst way, because the stage-two rescan through BlockingResidual reports no findings and the header pattern still fires, so the store records that a PEM secret was handled while a fragment of the key remains in the text. The argument the change rested on, that no canonically rendered body line falls under forty characters because RFC 7468 mandates sixty-four and OpenSSH writes seventy, is true only of an untruncated and un-rewrapped block, and a truncated or narrow-rewrapped paste is precisely the shape a transcript redactor exists to catch. Three independent adversarial reviews found this, one grading it critical. The fix must restore the lost shape without restoring the defect the raised bar was closing, which was a mentioned header opening a block on any sixteen-character word: the pair rule already distinguishes a wrapped body from prose, so the single-line path needs a discriminator that is not width alone. Detector: a header followed by one body line of thirty-two base64 characters and then a prose line must redact that body line, the two-consecutive-narrow-lines case must keep redacting, and the mentioned-header-plus-fenced-identifier case must still be left untouched.

## Grounds

- pursued: the reported shapes redact again while every prose case the raised bar protects stays byte-identical; a prose line consumed by the narrow rule, or a truncated body still surviving under a pasted header, would show it wrong
