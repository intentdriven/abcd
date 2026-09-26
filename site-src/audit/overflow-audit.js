// CI-ONLY. This file is never served, never bundled, and never emitted into the
// site output.
//
// Nothing under `site-src/audit/` can reach the built tree: the build copies
// site-src by an explicit allowlist — `copiedSources` in internal/core/site/
// build.go names exactly five files (redirects, headers, site.css, site.js,
// record.js) — and there is no directory walk over site-src anywhere in the
// build. Adding a file here therefore cannot change what visitors receive. If
// that allowlist is ever replaced by a directory-wide copy, this directory must
// be excluded in the same change.
//
// # What this is
//
// The RENDERED half of the mobile audit that itd-135 AC 7, itd-136 AC 8 and
// itd-137 AC 7 require: a real browser loads every route family at four
// viewport widths and fails if the document scrolls horizontally.
//
// `abcd lint site`'s mobile gate is the STATIC half — it reads the emitted
// HTML and CSS for a viewport meta, overflow containers, image constraints and
// inline widths wider than 390px. That gate runs in the build path and blocks a
// release; it cannot see layout, because nothing has been laid out. This one
// measures the laid-out document, and runs only in an optional CI job that
// gates nothing (adr-47 permits Playwright here and nowhere else).
//
// # Usage
//
//   node site-src/audit/overflow-audit.js <base-url> <screenshot-dir>
//
// Exits 0 when no page overflows at any width in any scheme, 1 otherwise, and
// writes a PNG per route/width/scheme either way — a failure is most useful
// with the picture beside it.

'use strict';

const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

// The four widths the record names: two phones, a tablet, a laptop.
const WIDTHS = [360, 390, 768, 1360];

// Two contexts, chosen to exercise the two media preferences the record
// requires alongside the geometry rather than in a separate pass: a page that
// only overflows in dark mode is still a page that overflows.
const SCHEMES = [
  { name: 'light-motion', colorScheme: 'light', reducedMotion: 'no-preference' },
  { name: 'dark-reduced', colorScheme: 'dark', reducedMotion: 'reduce' },
];

// One pixel of tolerance. Sub-pixel layout rounds scrollWidth up on perfectly
// legitimate pages, and a gate that cries wolf on rounding is a gate people
// learn to ignore. Anything a reader could actually scroll to is wider than this.
const TOLERANCE_PX = 1;

// The fixed route families. Record pages are discovered below rather than
// listed, so a renamed or retired record cannot leave this file pointing at a
// 404 that reports as a pass.
const ROUTE_FAMILIES = [
  '/',
  '/record/',
  '/record/graph/',
  '/record/foundations/',
  '/record/development/',
  '/record/health/',
  '/contributors/',
  '/references/',
];

// One representative page per record type that has its own page shape.
const SAMPLED_TYPES = ['adr', 'intent', 'issue'];

function slug(route) {
  const s = route.replace(/^\/+|\/+$/g, '').replace(/[^A-Za-z0-9._-]+/g, '-');
  return s === '' ? 'index' : s;
}

// discoverRecordRoutes reads the record export the build emits and picks the
// lowest-numbered record of each sampled type. Lowest-numbered, not random: the
// audit must compare like with like from run to run, or a screenshot diff is
// noise.
async function discoverRecordRoutes(page, baseUrl) {
  const res = await page.request.get(`${baseUrl}/record.json`);
  if (!res.ok()) {
    throw new Error(`record.json is not served at ${baseUrl} (HTTP ${res.status()})`);
  }
  const data = await res.json();
  const nodes = Array.isArray(data.nodes) ? data.nodes : [];
  const routes = [];
  for (const type of SAMPLED_TYPES) {
    const of = nodes
      .filter((n) => n && n.type === type && typeof n.id === 'string')
      .sort((a, b) => {
        const na = parseInt(a.id.replace(/\D+/g, ''), 10);
        const nb = parseInt(b.id.replace(/\D+/g, ''), 10);
        return na - nb;
      });
    if (of.length === 0) {
      throw new Error(`record.json holds no node of type '${type}'; the audit cannot sample one`);
    }
    routes.push(`/record/${type}/${of[0].id}/`);
  }
  return routes;
}

// measure returns the widest overflow on the laid-out page, and names what
// caused it. A bare "this page overflows" sends someone hunting; an element and
// its class is a place to look.
async function measure(page, viewportWidth, tolerance) {
  return page.evaluate(
    ({ width, tol }) => {
      const de = document.documentElement;
      const culprits = [];
      if (de.scrollWidth > width + tol) {
        for (const el of document.querySelectorAll('body *')) {
          const r = el.getBoundingClientRect();
          if (r.width === 0 && r.height === 0) continue;
          if (r.right > width + tol || r.left < -tol) {
            culprits.push({
              tag: el.tagName.toLowerCase(),
              cls: String(el.className || '').slice(0, 60),
              left: Math.round(r.left),
              right: Math.round(r.right),
            });
            if (culprits.length >= 5) break;
          }
        }
      }
      return { scrollWidth: de.scrollWidth, clientWidth: de.clientWidth, culprits };
    },
    { width: viewportWidth, tol: tolerance }
  );
}

async function settle(page) {
  await page.waitForLoadState('load');
  // The site makes no network calls at runtime (adr-38, adr-48: everything is
  // injected at build time), so there is no request to wait on — but web fonts
  // change metrics, and the chart lays itself out from record.js. Wait for the
  // fonts, then give the layout a beat. Animations are off in the reduced-motion
  // context, and the light context has nothing long-running to outrun.
  await page.evaluate(() => (document.fonts ? document.fonts.ready : null));
  await page.waitForTimeout(250);
}

async function main() {
  const baseUrl = (process.argv[2] || '').replace(/\/+$/, '');
  const shotDir = process.argv[3];
  if (!baseUrl || !shotDir) {
    console.error('usage: node overflow-audit.js <base-url> <screenshot-dir>');
    process.exit(2);
  }

  const browser = await chromium.launch();
  const failures = [];
  let checked = 0;

  try {
    const probe = await browser.newContext();
    const probePage = await probe.newPage();
    const routes = ROUTE_FAMILIES.concat(await discoverRecordRoutes(probePage, baseUrl));
    await probe.close();

    console.log(`auditing ${routes.length} routes at ${WIDTHS.join('/')} px in ${SCHEMES.length} schemes`);
    for (const route of routes) console.log(`  ${route}`);

    for (const scheme of SCHEMES) {
      for (const width of WIDTHS) {
        const context = await browser.newContext({
          viewport: { width, height: 900 },
          colorScheme: scheme.colorScheme,
          reducedMotion: scheme.reducedMotion,
          deviceScaleFactor: 1,
        });
        const page = await context.newPage();
        for (const route of routes) {
          const url = `${baseUrl}${route}`;
          const res = await page.goto(url, { waitUntil: 'load' });
          if (!res || !res.ok()) {
            failures.push(`${route} [${scheme.name} ${width}px] did not load (HTTP ${res ? res.status() : 'none'})`);
            continue;
          }
          await settle(page);

          const dir = path.join(shotDir, scheme.name, String(width));
          fs.mkdirSync(dir, { recursive: true });
          await page.screenshot({ path: path.join(dir, `${slug(route)}.png`), fullPage: true });

          const m = await measure(page, width, TOLERANCE_PX);
          checked += 1;
          if (m.scrollWidth > width + TOLERANCE_PX) {
            const who = m.culprits
              .map((c) => `<${c.tag}${c.cls ? ` class="${c.cls}"` : ''}> spans ${c.left}..${c.right}`)
              .join('; ');
            failures.push(
              `${route} [${scheme.name} ${width}px] scrollWidth ${m.scrollWidth} > ${width}` +
                (who ? ` — ${who}` : '')
            );
          }
        }
        await context.close();
      }
    }
  } finally {
    await browser.close();
  }

  console.log(`\n${checked} page renderings measured`);
  if (failures.length > 0) {
    console.log(`\n${failures.length} overflowing:`);
    for (const f of failures) console.log(`  ${f}`);
    // A GitHub annotation, so the failure is readable from the run summary
    // without opening the log.
    console.log(`::error::${failures.length} page rendering(s) scroll horizontally; see the screenshots artifact`);
    process.exit(1);
  }
  console.log('no page scrolls horizontally at any audited width.');
}

main().catch((err) => {
  console.error(err && err.stack ? err.stack : String(err));
  process.exit(1);
});
