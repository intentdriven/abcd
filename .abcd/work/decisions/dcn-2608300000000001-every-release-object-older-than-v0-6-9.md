---
id: dcn-2608300000000001
slug: every-release-object-older-than-v0-6-9
date: 2026-08-30
---
2026-08-30 — Every release object older than v0.6.9 is deleted from the forge (v0.1.0, v0.2.0, v0.3.0, v0.4.2, v0.5.1, v0.6.7, v0.6.8; the v0.6.0–v0.6.6 tags never had one), and every tag is kept. The three v0.6.9 advisories mark `<= 0.6.8` vulnerable, and a release object is the only thing that makes a vulnerable binary downloadable by version: the installer, the bootstrap hook and `abcd update` all resolve latest, so removing old assets affects only a deliberate pin, which is the point. Tags are the immutable audit trail the release gate relies on (adr-52), and the build-provenance and receipt attestations that named those releases stay in the attestation store even though the artefacts they attest are gone; the deletion is recorded here because no earlier release removed a predecessor and the decision log carried no practice for it. Reversing it means re-cutting a release from the surviving tag, never re-uploading old assets by hand.
