---
schema_version: 1
id: "iss-2608221342502032"
slug: "site-tar-gz-is-attested-but-absent-from"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "agent-finding"
found_at: ".github/workflows/site.yml"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed G): Should checksums.txt cover site.tar.gz, or is the manifest binaries-only by design?"
remedy: "Waits on ruling G (checksums cover site.tar.gz, or binaries-only): if covered, build site.tar.gz before release.yml's checksums step so checksums.txt hashes it, never appending to a manifest already attested and published; if binaries-only, state that scope in release.yml's header comment and the install docs, naming the attestation as site.tar.gz's integrity check. Either way, pin the manifest's asset list in a test so a new asset cannot fall outside it silently."
---

site.tar.gz is attested but absent from checksums.txt, so the release manifest no longer covers every asset; wants a deliberate decision rather than a silent gap

## Remedy grounds (2026-09-29)

- GoReleaser's default checksum file covers all published binaries, archives, packages and source archives, with extra_files to add more: https://goreleaser.com/customization/checksum/ (consulted 2026-09-29), so full coverage is the common default; the owed ruling decides whether abcd follows it.
- Rejected: rewriting checksums.txt after publication, which would invalidate the attestation already made over it.
