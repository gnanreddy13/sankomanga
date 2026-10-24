# Design decisions

The look of the app, as decided in conversation. These are starting points,
not rules: the design can change at any time.

## Overall

| | Decision |
|---|---|
| Feel | Dark: near-black background, covers are the main visual. Minimal, aesthetic, premium. |
| Accent colour | One quiet neutral accent on home and results. On a title page, accents (score, links, glow) use that title's `coverColor`, falling back to the neutral accent if it's missing or too dark to read. |
| Fonts | One clean sans-serif (Inter) for everything; titles are just larger and bolder. Japanese text falls back to the device's Japanese font. |
| Theme | Dark only for now (no light-mode switch). |
| Footer | "Data from AniList" (linking to anilist.co) on every page, as AniList's terms require. |

## Home screen

One centred, width-limited box:

| Part | Decision |
|---|---|
| Left (about 55%) | 2–3 columns of popular-manga covers (from `/api/showcase`, ecchi excluded), drifting slowly up and down in opposite directions, endlessly, with faded top and bottom edges. Purely decorative: no hover or click. |
| Top right | App name, with the tagline below |
| Bottom right | Search bar, lined up with the bottom of the covers. Enter goes to `/search?q=...`. |
| Phones | Stacked: name + tagline, search, then a slowly moving strip of covers |
| Motion | Still covers when the device has "reduce motion" on; paused when the tab is hidden |
| Loading / failure | Search works immediately; covers fade in (each shows its own colour while loading); a soft gradient if they can't load |

App name: placeholder `manga-app`. Tagline: "explore your next manga".
Both live in one place each (`web/js/config.js`).

## Results screen

- Slim top bar: app name on the left (links home), compact search bar.
- Grid of cards. Medium covers (about 160px): ~2 per row on phones, 4 on
  tablets, 5–6 on a laptop.
- Card: cover, title (up to 2 lines), one small grey line like
  `2009 · Manga · ★ 8.4`. Missing year or score is left out of the line.
- Grey placeholder cards while loading. Covers lazy-load and show their
  `coverColor` until the image arrives.
- No results: "No manga found for 'xyz'" with a hint to check the spelling.

## Title page

- Wider screens: big cover on the left, details on the right (title,
  subtitle, facts row: format · status · year · chapters · volumes, score as
  `★ 8.4`, genres as small rounded tags, "View on AniList" link), synopsis
  below. Phones: stacked, cover first.
- A soft glow in the cover's colour behind the cover.
- Missing facts are left out, never shown as "—".

## Errors

Friendly wording, e.g. "AniList is busy right now, try again in a few seconds."

## Frontend approach

Plain HTML, CSS and JavaScript, no framework and no build step. Each screen
is its own HTML page with one small script; Go serves them together with the
API (so no CORS is needed) and bundles them into the binary with `embed`.
Visual design is not limited by this: anything CSS can do is possible, and
libraries can be added for specific effects if ever needed.
