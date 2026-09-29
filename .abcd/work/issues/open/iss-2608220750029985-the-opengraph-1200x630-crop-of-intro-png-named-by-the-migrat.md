---
schema_version: 1
id: "iss-2608220750029985"
slug: "the-opengraph-1200x630-crop-of-intro-png-named-by-the-migrat"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "agent-observation"
found_at: "docs/assets/img"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker: spc-37 lists the OpenGraph crop as out of scope and a product decision, and the site emits no og: meta, so a committed crop would be dead weight until someone chooses the picture and wants the social card (drain lane drainRest, run A, 2026-09-29)."
remedy: "Waits on the social-card ruling (spc-37 leaves it a product decision): if wanted: commit a 1200x630 crop of intro.png under docs/assets/img and have the landing page emit og:title, og:type, og:image (absolute URL) and og:url with og:image:width, og:image:height and og:image:alt, proven by a site-render test on the four required properties and the image's dimensions; if not wanted: wontfix the record, since a crop with no og: meta is dead weight."
---

the OpenGraph 1200x630 crop of intro.png named by the migration map is not yet created or committed; the landing page ships without a social card until the asset lands

## Remedy grounds (2026-09-29)

- The Open Graph protocol requires og:title, og:type, og:image and og:url and defines the image's width, height and alt properties, but sets no size (https://ogp.me/, checked 2026-09-29); 1200x630 at 1.91:1 is the consuming platform's recommendation (https://developers.facebook.com/docs/sharing/webmasters/images/, checked 2026-09-29).
- Rejected: committing the crop alone, which the deferral already calls dead weight.
