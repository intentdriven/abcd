---
schema_version: 1
id: "iss-2608261041218890"
slug: "release-yml-tag-not-shape-checked-before-make-build"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-a/round-7"
found_at: ".github/workflows/release.yml"
resolution: "release.yml and the scaffold template (both profiles) shape-check TAG with the site.yml [[ =~ ]] vX.Y.Z pattern as the release job's first step after checkout, before the build, the archive render and the Release creation. TestReleaseShapeChecksTheTagBeforeBuilding runs the step against the reproduced injection tag and others."
impact: fix
resolved_by:
  commit: "da294066873d033e67b407e91fa165f212e3fa82"
---

release.yml feeds an unvalidated tag name into 'make build VERSION=$TAG', splicing it textually into the -ldflags shell recipe. TAG is inputs.tag or github.ref_name with no shape-check; the release job runs 'make build VERSION="${TAG}"' and Make substitutes $(VERSION) textually into 'go build -ldflags "...Version=$(VERSION)"', so a tag containing a double-quote closes the quoting and splices a shell command (reproduced: VERSION='v9.9.9";echo INJECTED;x="'). Such a tag is a legal git ref matching the v* trigger. site.yml shape-checks the identical value with a [[ =~ ]] vX.Y.Z pattern and documents why; release.yml shape-checks the sha but not the tag. Bounded: the injectable step is behind the release environment's required reviewer and github.ref_name is settable only by an insider who can already run code in CI, so it grants no new capability — defense-in-depth, but a real shape-check-parity gap that also propagates to the scaffold template (release.yml.tmpl). RECORDED NOT FIXED this round: release.yml cannot be exercised by CI, so an autonomous edit to the release pipeline is deferred to the maintainer. Proposed fix: add a tag=$TAG shape-check step (the site.yml regex, [[ =~ ]] not grep) before make build, in both release.yml and the scaffold template.

## Grounds

- pursued: a v* tag carrying a quote, a newline or no X.Y.Z core is refused before any step consumes it; shown wrong by a release job reaching make build with such a tag
