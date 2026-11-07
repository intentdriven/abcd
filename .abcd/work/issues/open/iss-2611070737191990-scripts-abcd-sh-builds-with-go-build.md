---
schema_version: 1
id: "iss-2611070737191990"
slug: "scripts-abcd-sh-builds-with-go-build"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/abcd.sh"
remedy: "Pass -trimpath in scripts/abcd.sh and the narrow-run template, then delete the default-branch Go cache in CI so the nightly saves it in the new form."
---

scripts/abcd.sh builds with go build -buildvcs=false but without -trimpath, so the same source in a second checkout gets its own build-cache entries; the preflight reuse stage already uses -trimpath. Builders' narrow runs behave the same, and a second checkout's build took 170 MiB and 6 s against 1.9 MiB and 1 s with the flag.
