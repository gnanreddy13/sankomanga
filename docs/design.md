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
| Footer | "Data from" followed by AniList's official logo (18px, its original colours, loaded from `anilist.co/img/icons/icon.svg`, alt text "AniList"), linking to anilist.co, on every page. AniList's terms only regulate the AniList *name* in app names; they say nothing about the logo. Never recolour the logo. |

## Home screen

No box: the content sits directly on the page, centred.

| Part | Decision |
|---|---|
| Background | Pure black (`#000`). |
| Size | Everything kept small and quiet. |
| Left | A tall, narrow window (about 250–310px wide, as tall as the right side) of popular covers (from `/api/showcase`, ecchi excluded) drifting up and down in three columns, endlessly: each column at its own slow speed, the middle one the other way. No box: the window fades into the black at the top and bottom (the sides stay crisp), and has a slight 3D tilt towards the text. A soft glow behind it slowly cycles through the covers' colours. Purely decorative: no hover or click. |
| Right | Name and search grouped together in the vertical middle beside the covers. The app name and tagline, then the search below. (A short hairline rule under the tagline was tried and removed.) |
| Tagline | Italic, muted grey. |
| Search | Underline search, same style on both pages: a single grey line ("Search a title…"), with a small magnifying-glass button on the right (both pages). The button just turns white on hover (no backing). A lighter grey part stretches full width while typing. Enter goes to `/search?q=...`. |
| Phones | Stacked and centred: a band of drifting covers (no tilt), then name + tagline, then search |
| Motion | With "reduce motion" on: still covers, one fixed colour, no entrance animation. Drifting pauses while the tab is hidden. |
| Loading / failure | Search works immediately; covers fade in (each shows its own colour while loading); a soft gradient if they can't load |

App name: placeholder `manga-app`. Tagline: "explore your next manga".
Both live in one place each (`web/static/js/config.js`). Logo: not decided yet.

Tried and dropped for the home page (kept here so we don't repeat them):
- A box around the content.
- A 3D flip book (three.js) riffling through covers to the logo: impressive,
  but heavy, played only once, and the page backs never looked right
  (real interior pages would need rights we don't have).
- A page counter ("01 / 15") above the name.

## Results screen

Centred, like the home page, on the same pure black.

| Part | Decision |
|---|---|
| Header | Lined up with the left edge of the grid: "back to [home icon]" (small grey text and icon that simply turn white on hover; links home); then a smaller version of the underline search holding the current search term; then the count in small italic grey ("6 results", or "20+ results" when there are more pages, since the API has no total). |
| Grid | A centred block about 800px wide. Balanced covers (~140px): 5 a row on laptops, 4 on tablets, 3 on phones (~105px). |
| Card | Cover, title (up to 2 lines, then "…"), and one small dim info line like `2009 · Manga · ★ 8.4` (missing year or score left out; on phones the line only wraps between parts). The whole card links to `/manga/{id}`. |
| Hover | The cover lifts a few pixels with a soft white glow behind it (the same colour for every cover); the title brightens. Same on keyboard focus. |
| Loading | Header right away; 20 dark placeholder cards with a slow, faint shimmer. Covers lazy-load and show their dimmed `coverColor` until the image fades in. |
| No results | In the count's place: "No manga found for "xyz". Check the spelling, or try the English or Japanese title." |
| More than 20 | Under the grid: "Showing the 20 best matches. Try a more specific title to narrow it down." ("Load more" is a later feature.) |
| Errors | One calm line instead of the grid: AniList busy (502/503), visitor rate-limited (429), or "Something went wrong" for anything else. |
| Motion | With "reduce motion" on: no lift, no shimmer. |

Tried and dropped: a slim top bar with the name and search side by side;
the search term as a big centred headline over an underline; a centred
soft grey pill with a grain texture; a grey-outlined pill with a
magnifying glass on both pages (the underline search won).

## Title page

A centred block (about 780px wide), centred vertically in the window too
(like the home page; long pages simply start at the top and scroll), with
"back to [search icon]" above it, lined up with its left edge. It goes
back to the same search results (never a fresh search): browser "back"
when coming from the results page (keeps the scroll position), otherwise
the results for the search carried in the address (`/manga/53390?q=berserk`,
added by the result cards), or home when there was no search.

| Part | Decision |
|---|---|
| Layout | Cover beside details: the cover on the left, everything else on the right. Phones: stacked, the cover (~180px) centred on top. |
| Cover | About 240px wide, with a soft white glow behind it (same as the results hover). Shows its dimmed own colour until the image fades in. |
| Title | Large, then the subtitle in italic grey (hidden when there isn't one). |
| Facts | One quiet grey line, e.g. `Manga · Finished · 2009 · 141 chapters · 34 volumes`. Missing facts drop out (an ongoing series reads `Manga · Releasing · 1997`). Wraps only between parts. |
| Score | On its own line, larger and white: `★ 8.4`. Hidden when unknown. |
| Genres | Small outlined grey pills. Not clickable (yet). |
| Synopsis | Our cleaned paragraphs, 15px, light grey, line height 1.7, at most ~60 characters a line. Notes look like any other paragraph. |
| AniList link | "View on AniList ↗" under the synopsis, small and grey, opens in a new tab. |
| Browser tab | "Shingeki no Kyojin · manga-app". |
| Loading | Shimmering cover placeholder and faint bars for the text. |
| Not found | For unknown or filtered ids (and non-numeric ones): "This title isn't available here. Search for another" (links home). |
| Errors | The same one-liners as the results page. |

Missing facts are always left out, never shown as "—".

## Errors

Friendly wording, e.g. "AniList is busy right now, try again in a few seconds."

## Frontend approach

Plain HTML, CSS and JavaScript, no framework and no build step. Each screen
is its own HTML page with one small script; Go serves them together with the
API (so no CORS is needed) and bundles them into the binary with `embed`.
Visual design is not limited by this: anything CSS can do is possible, and
libraries can be added for specific effects if ever needed.

Outside resources in use: only the Inter font, from Google Fonts. It could
be self-hosted before deploying, so visitors' browsers don't contact a third
party.
