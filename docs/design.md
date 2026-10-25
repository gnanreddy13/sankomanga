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
| Tab icon | サ (the first sound of サンコー) in white on a black rounded square, drawn as three strokes so it looks the same on every device. One file, `web/favicon.svg`, served as `/favicon.ico` and linked from every page. |
| Loading | The home page tells the browser up front about the scripts it will need and the covers list (preload hints), so it fetches them all at once instead of one after another. The server keeps the covers list ready, so it answers instantly. |
| Page changes | A soft 0.25s crossfade between pages of the site (the browser's cross-page view transitions: Chrome, Edge and Safari; other browsers change pages as before). Off with "reduce motion". |
| Footer | "Data from" followed by AniList's official logo (18px, its original colours, loaded from `anilist.co/img/icons/icon.svg`, alt text "AniList"), linking to anilist.co, on every page (on the home page it sits in the middle content instead of a footer). AniList's terms only regulate the AniList *name* in app names; they say nothing about the logo. Never recolour the logo. |

## Home screen

No box: the content sits directly on the page. The same layout on every
screen size: a row of popular covers along the top, the name, tagline and
search hint centred in the middle, and another row of covers along the
bottom.

| Part | Decision |
|---|---|
| Background | Pure black (`#000`). |
| Small text | The faintest grey is `#77777f` (4.7:1 contrast on black, above the 4.5:1 small text needs). Never darker. |
| Size | Everything kept small and quiet. |
| Covers (top and bottom) | Two rows of popular covers (from `/api/showcase`: the 30 most popular, ecchi excluded, shuffled on every visit). To keep the page light, each cover's image downloads only when the cover glides within 300px of the screen (about 18 at first, then one every few seconds), and the browser picks the size: 230px-wide images on ordinary screens, 460px on sharp "Retina" screens. Each row gets its own 15, so every cover on the screen is different (only screens wider than about 3,000px would repeat a cover within a row). Each cover has a soft white glow behind it, the same for every cover (like the results and title page glows). Each row glides endlessly and slowly (22px a second): the top one to the left, the bottom one to the right. Covers are about 280px tall on a laptop (at most 30% of the screen height). On phones they take whatever height the middle content doesn't need (at most 340px: about 160px on small phones, 220px on a standard one, 270px on the largest), with about 55px between the covers and the middle (a little less on short phones: about 46px at 667px tall, 38px at 568px). Phones turned sideways (and any screen under 500px tall) work the same way: the name stays on one line but is sized by the height too (at most 17% of it), so the covers keep about 85px and the gaps about 26px on a typical sideways phone. After a rotation or window resize, the rows are rebuilt for the new size (once it settles), so the loop never shows a gap, with 14px gaps (10px on phones) and 8px rounded corners. Both ends of each row fade softly into the black. Purely decorative: no hover or click. |
| Wordmark (middle) | On one line on wider screens: the name in Inter 900 capitals (96px). On phones it's stacked, SANKO over MANGA, much bigger (SANKO about 80px, 22% of the screen width), with MANGA's art stretched to exactly the width of the SANKO line, so the art is about twice as big as it would be on one line. SANKO is plain white, taking turns with サンコー (the Japanese touch; it reads "sankō", like 参考, "reference"; extra-heavy Noto Sans JP, set a little smaller to match the capitals' height, only its four characters downloaded). The swap goes sound by sound, left to right, each 130ms after the one before: SA becomes サ, N becomes ン, KO becomes コー, so the name visibly translates itself (`NAME_SOUNDS` in `config.js`). Each sound leaving drifts up, blurs and fades while its katakana settles in from below (0.7s). SANKO shows for 6 seconds (including before the first swap, so people read it first), サンコー for 3: the English name is the one people need to remember. Both share one spot, as wide as the wider of the two, so MANGA never moves. With "reduce motion" on, it stays SANKO; swapping pauses while the tab is hidden. Then MANGA as art: an image of the word with black-and-white manga art inside each letter (`web/static/img/manga-art.png`, transparent around the letters, cropped tight to them), as tall as the capitals and on the same baseline, with a thin white outline traced around the letters (their own black outlines vanish on black). Its exact space is kept from the very first moment (the image's size is written in `home.js`, next to its path; update it if the image changes), so nothing moves when it arrives: the space stays empty while it downloads, then the art fades in over 0.6s (no fade with "reduce motion"; no fade if the browser already has it). If the file is missing, the styled letters below are shown instead. Without the image, MANGA shows as outlined capitals filled with fine white screentone dots. The art must be original before the site goes public (no existing series' characters); the current file is a private preview and is git-ignored. Screen readers hear just "sankomanga"; the katakana is decoration. Other pages keep the plain lowercase name. |
| Below the wordmark | The tagline, centred. Then, 24px below, a caption row exactly as wide as the name, like the caption under a magazine title: the search hint against the name's left edge, "Data from [AniList logo]" against its right edge, both in small (11px), spaced-out, faint capitals, with a 14px logo. The home page has no footer; the other pages keep theirs. No search bar on the page. (A short hairline rule under the tagline was tried and removed.) |
| Tagline | Italic, muted grey. |
| Search hint | Small faint text that's also a real button: "Type to search" on devices with a keyboard and mouse, "Tap to search" on touch screens. |
| Search overlay | The search box opens on top of the page, which is dimmed and blurred. It opens by typing any letter (that letter starts the search), "/", clicking or tapping anywhere (except links), or the hint. It sits in the middle of the screen, where the name is (just clear of phone keyboards), in a bigger version of the underline style. Closes with Esc (the first Esc clears any text), a click/tap outside, or the browser's back button/gesture (it adds one history step while open; searching replaces that step, so "back" from the results goes straight home). Tab stays inside the box while open. Opens with a 0.25s fade (the box rises slightly); closes with the reverse, a 0.2s fade (the box sinks slightly). The page is usable as soon as closing starts, and typing or clicking during the fade reopens it at once. No fades with "reduce motion". While it's open, the cover rows and the SANKO ↔ サンコー swap pause (moving covers under the blur looked like they were shimmering); they carry on from the same spot when it closes. |
| Search style | Underline search, same style in the home overlay and on the results page: a single, steady grey line (no animation), placeholder "Search…", with a small magnifying-glass button on the right (both pages). The line turns a lighter grey while the box is selected (so keyboard users can see where they are). The button just turns white on hover (no backing). While there's text, a small × appears just left of it: it clears the box and keeps the cursor there (Escape does the same). Custom rather than the browser's own clear button, which looks different in every browser. Enter goes to `/search?q=...`. |
| Motion | With "reduce motion" on: still rows, no entrance animation. Gliding pauses while the tab is hidden. |
| Loading / failure | Search (the overlay) works immediately; the rows fade in once the list arrives, and each cover shows its own colour until its image loads. If the covers can't load, the rows simply stay empty. |

App name: **sankomanga** (the domain will be English too). Tagline: "explore
your next manga". Both live in one place each (`web/static/js/config.js`,
plus `appName` in `config.go` for the backend), as do the art-filled word
(`ART_WORD`) and the Japanese name (`JAPANESE_NAME`). The name is written
lowercase everywhere except the home page wordmark, which is set in capitals
(a logo can be styled differently from the name in text). Logo: the home
page wordmark for now.

Tried and dropped for the home page (kept here so we don't repeat them):
- A box around the content.
- A 3D flip book (three.js) riffling through covers to the logo: impressive,
  but heavy, played only once, and the page backs never looked right
  (real interior pages would need rights we don't have).
- A page counter ("01 / 15") above the name.
- On desktop, the covers on the left (slightly tilted towards the text) with
  the name and search beside them; now stacked like on phones everywhere.
- A visible search bar on the home page (replaced by the hint and overlay).
- Covers on top in a narrow window, drifting up and down in three columns,
  with the name below (replaced by the fan, which uses the whole width).
- A fan of 7 covers rising from the bottom edge, breathing and swapping one
  cover every 4 seconds, with a glow in the newest cover's colour (replaced
  by the two gliding rows).
- A different cover inside each letter of "manga" (lowercase): colourful
  covers in small lowercase letters looked busy; replaced by black-and-white
  manga art in capital MANGA.
- The hint and the credit as two separate centred lines under the tagline:
  next to the huge name they looked like leftovers (replaced by the caption
  row).
- SANKO and サンコー crossfading as whole words, 4 seconds each: smooth but
  generic; replaced by the sound-by-sound swap.
- サンコー as a huge, very dark watermark behind the name (replaced by
  SANKO and サンコー taking turns in the name itself).
- A manga page texture drawn in CSS inside MANGA (screentone, speed lines,
  hatching, panel gutters): at this size it read as stripes, like a glitch.

## Results screen

Centred, like the home page, on the same pure black.

| Part | Decision |
|---|---|
| Header | Lined up with the left edge of the grid: "back to [home icon]" (small grey text and icon that simply turn white on hover; links home); then a smaller version of the underline search holding the current search term ("/" jumps to it and selects the text) (280px on desktop; full width on phones, lining up with both edges of the grid); then the count in small italic grey: "6 results", or "20 closest matches" when there are more pages (no grand total: AniList reports 5000 for every search). Kept to what the cards can't tell you. |
| Grid | A centred block about 800px wide. Balanced covers (~140px): 5 a row on laptops, 4 on tablets, 3 on phones (~105px). |
| Card | Cover, title (up to 2 lines, then "…"), and one small dim info line like `2009 · Manga · Ongoing · ★ 8.4` ("Ongoing" only for titles still being published; missing year or score left out; on phones the line only wraps between parts). The whole card links to `/manga/{id}`. |
| Hover | Only on devices with a real pointer (on touch screens a tap would leave it stuck). The cover lifts a few pixels with a white glow behind it (the same colour for every cover): a glow spreading below plus a faint halo, like the title page cover but a little softer so it doesn't spill onto neighbouring covers. The title brightens. Same on keyboard focus. |
| Loading | Header right away; 20 dark placeholder cards with a slow, faint shimmer. Covers lazy-load and show their dimmed `coverColor` until the image fades in. |
| No results | In the count's place: "No manga found for "xyz". Check the spelling, or try the English or Japanese title." |
| More than 20 | Under the grid: "Only the 20 closest matches are shown. Try a more specific title." ("Load more" is a later feature.) |
| Errors | One calm line instead of the grid: AniList busy (502/503), visitor rate-limited (429), or "Something went wrong" for anything else. |
| Motion | With "reduce motion" on: no lift, no shimmer. |

Tried and dropped: a summary line totalling formats and ongoing titles
("4 manga, 2 one-shots · 2 ongoing": it repeated what the cards say, and
didn't say which were ongoing); a slim top bar with the name and search side by side;
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
| Cover | About 240px wide, with a strong white glow behind it (the same colour for every title; stronger than the results hover glow): a soft glow spreading below, plus a faint halo around the edges. Shows its dimmed own colour until the image fades in. |
| Title | Large, then the subtitle in italic grey (hidden when there isn't one). |
| Facts | One quiet grey line, e.g. `Manga · Finished · 2009 · 141 chapters · 34 volumes`. Missing facts drop out (an ongoing series reads `Manga · Ongoing · 1997`). Wraps only between parts. |
| Score | On its own line, larger and white: `★ 8.4`. Hidden when unknown. |
| Genres | Small outlined grey pills. Not clickable (yet). |
| Synopsis | Our cleaned paragraphs, 15px, light grey, line height 1.7, at most ~60 characters a line. Notes look like any other paragraph. |
| AniList link | "View on AniList ↗" under the synopsis, small and grey, opens in a new tab. |
| Browser tab | "Shingeki no Kyojin · sankomanga". |
| Loading | Shimmering cover placeholder and faint bars for the text. |
| Status words | AniList's RELEASING is shown as "Ongoing" everywhere (cards and title page): clearer than "Releasing". |
| Not found | For unknown or filtered ids (and non-numeric ones): "This title isn't available here. Search for another" (links home). |
| Errors | The same one-liners as the results page. |

Missing facts are always left out, never shown as "—".

## Keyboard and touch

- Keyboard: everything is reachable with Tab in reading order, with visible
  focus; Enter searches and opens cards; Esc clears / closes; "/" opens the
  search (home) or jumps to it (results).
- Touch: small icon buttons get invisible tap areas of at least 44px on touch
  screens (no visual change). Covers dip slightly while pressed. Hover effects
  only apply on devices with a real pointer. No custom swipe gestures: the
  browser's own back gesture works everywhere.

## Page not found

Any address that isn't a page of the site gets a small centred message
("Page not found" / "There's nothing at this address." / "back to [home]")
with a real 404 status. `/favicon.ico` serves the site's icon.

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
