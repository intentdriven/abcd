---
schema_version: 1
id: "iss-2609261737325810"
slug: "review-of-integ4-reported-that-abcd-capture-defer-prints"
severity: "nitpick"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-integ4"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
wontfix_reason: "By design: the line is the checkout banner iss-2609202053570475 added so a ledger verb names the checkout it addressed. It goes to stderr, never stdout, and fsutil.RedactHome makes it home-relative, so under the real HOME it reads ~/... Routing it through scrubPaths would also turn the working directory into '.', which erases the identity the banner exists to give. A checkout outside HOME carries no developer-identity root, the same stance scrubPaths documents. Sibling sweep: capture bare, list, promote, link, disposition, wontfix, defer and the record dispatcher print only this banner, and --json only its ledger member."
---

Review of integ4 reported that abcd capture defer prints 'ledger of <absolute cwd>' on stdout unscrubbed. Reproduced: the line is captureLedgerRoot's checkout banner (iss-2609202053570475), on stderr, printed by every capture verb and by the record dispatcher; --json carries the same identity as its ledger member. It is home-redacted through fsutil.RedactHome, so under the real HOME it reads ~/..., and it is absolute only for a checkout outside HOME, which is how the review's temporary-HOME scratch run saw it.

## Grounds

- declined: the banner is home-redacted and on stderr by design; a capture verb printing the home directory, or any path on stdout outside --json's ledger member, would show this wrong
