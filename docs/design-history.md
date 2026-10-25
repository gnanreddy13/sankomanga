# Design history

The full log of design decisions made while building sankomanga, with the
reasoning, measurements and everything that was tried and dropped. Kept so
old ideas aren't repeated. Some entries describe earlier versions of the
site; for how it looks now, see [design.md](design.md).


## Overall

| | Decision |
|---|---|
| Feel | Dark: near-black background, covers are the main visual. Minimal, aesthetic, premium. |
| Accent colour | One quiet neutral accent on home and results. On a title page, accents (score, links, glow) use that title's `coverColor`, falling back to the neutral accent if it's missing or too dark to read. |
| Fonts | One clean sans-serif (Inter) for everything; titles are just larger and bolder. Japanese text falls back to the device's Japanese font. |
| Theme | Dark only for now (no light-mode switch). |
| Mini wordmark | On the results and title pages, top left: the home page's name shrunk to 17px: the same Inter 900 capitals, SANKO in solid white and MANGA as a 1px white outline. It links home and replaces the old "back to [home icon]" (also on the 404 page, centred under the message); screen readers hear "sankomanga, home". On hover and keyboard focus, MANGA's outline glows (a tight bright white halo plus a wider, fainter one, fading in over 0.25s); the letters stay hollow. Filled in from `APP_NAME` and `GREY_WORD` in `config.js`. |
| Tab icon | サ (the first sound of サンコー) in white on a black rounded square, drawn as three strokes so it looks the same on every device. One file, `web/favicon.svg`, served as `/favicon.ico` and linked from every page. |
| Loading | The home page tells the browser up front about the scripts it will need and the covers list (preload hints), so it fetches them all at once instead of one after another. The server keeps the covers list ready, so it answers instantly. |
| Link previews | When a link is shared in a chat app or social site: the name as the title, the meta description, and a 1200×630 picture (`web/og-image.png`: the SANKOMANGA wordmark and tagline on black). The server fills in the name and the site's full address in the tags, since those apps don't run JavaScript. |
| Home-screen icon | On iPhones (Add to Home Screen): the サ icon on black, 180×180 (`web/apple-touch-icon.png`); iOS rounds the corners itself. |
| Page changes | A soft 0.25s crossfade between pages of the site (the browser's cross-page view transitions: Chrome, Edge and Safari; other browsers change pages as before). Off with "reduce motion". |
| Footer | "Data from" followed by AniList's official logo (18px, its original colours, loaded from `anilist.co/img/icons/icon.svg`, alt text "AniList"), linking to anilist.co, on every page (on the home page it sits in the middle content instead of a footer). AniList's terms only regulate the AniList *name* in app names; they say nothing about the logo. Never recolour the logo. |

## Home screen

No box: the content sits directly on the page. Inspired by editorial
"hero" pages: lots of black, the name and tagline in the middle, a few
popular manga covers floating around them, and the small print
in the bottom corners.

| Part | Decision |
|---|---|
| Background | Pure black (`#000`). |
| Small text | The faintest grey is `#77777f` (4.7:1 contrast on black, above the 4.5:1 small text needs). Never darker. |
| Size | Everything kept small and quiet. |
| Covers (around the name) | Popular manga (from `/api/showcase`: the 30 most popular, ecchi excluded, shuffled on every visit), shown as flat covers floating around the name, like the covers on the other pages: 6px rounded corners, with a white glow (a halo close around each cover fading into a wider haze), about as bright as the title page cover's, so the site's glows feel like one family. 8 on laptops (about 110px wide), placed by hand around the name and never over it; 4 bigger ones on upright phones (two above, two below); 5 on phones turned sideways (sides and top). They stay alive quietly: each bobs 5px up and down at its own pace (7–9s); on desktops they shift a few pixels with the mouse, nearer ones more (depth); and every 5 seconds one cover dissolves into a different popular manga: the new cover is downloaded first, then fades in on top of the old one over 1.2s with an even ease-in-out, so the book never goes blank (each volume taking turns, never the same twice in a row). Covers load at the size that fits the screen (230px or 460px images), and hidden volumes on phones never download. Purely decorative: no hover or click. |
| Wordmark (middle) | On one line on wider screens: the name in Inter 900 capitals (64px). On phones it's stacked, SANKO over MANGA, bigger (SANKO about 56px, 15% of the screen width). SANKO is plain white, taking turns with サンコー (the Japanese touch; it reads "sankō", like 参考, "reference"; extra-heavy Noto Sans JP, set a little smaller to match the capitals' height, only its four characters downloaded). The swap goes sound by sound, left to right, each 130ms after the one before: SA becomes サ, N becomes ン, KO becomes コー, so the name visibly translates itself (`NAME_SOUNDS` in `config.js`). Each sound leaving drifts up, blurs and fades while its katakana settles in from below (0.7s). SANKO shows for 6 seconds (including before the first swap, so people read it first), サンコー for 3: the English name is the one people need to remember. Both share one spot, as wide as the wider of the two, so MANGA never moves. With "reduce motion" on, it stays SANKO; swapping pauses while the tab is hidden. Then MANGA as a white outline: hollow letters, the line about 1.5% of the font size (about 1px on a laptop, never under 1px). Inter draws letters from overlapping shapes, so the outline is drawn first and the letters are filled with the page black on top, hiding the inner joins. The small wordmark on the other pages uses the same outline at 1px. Screen readers hear just "sankomanga"; the katakana is decoration. The other pages show the same mark small (see "Mini wordmark"). |
| Below the wordmark | The tagline, centred. |
| Corners | The small print, in small (11px), spaced-out, faint capitals, 24px from the sides and 20px from the bottom: the search hint bottom-left, "Data from [AniList logo]" (14px logo) bottom-right. The home page has no footer; the other pages keep theirs. No search bar on the page. |
| Tagline | Italic, muted grey. |
| Search hint | Small faint text that's also a real button: "Type to search" on devices with a keyboard and mouse, "Tap to search" on touch screens. |
| Search overlay | The search box opens on top of the page, which is dimmed and blurred. It opens by typing any letter (that letter starts the search), "/", clicking or tapping anywhere (except links), or the hint. It sits in the middle of the screen, where the name is (just clear of phone keyboards), in a bigger version of the underline style. Closes with Esc (the first Esc clears any text), a click/tap outside, or the browser's back button/gesture (it adds one history step while open; searching replaces that step, so "back" from the results goes straight home). Tab stays inside the box while open. Opens with a 0.25s fade (the box rises slightly); closes with the reverse, a 0.2s fade (the box sinks slightly). The page is usable as soon as closing starts, and typing or clicking during the fade reopens it at once. No fades with "reduce motion". While it's open, the volumes' bob, the cover swaps and the SANKO ↔ サンコー swap pause (moving things under the blur look like they're shimmering); they carry on from the same spot when it closes. |
| Search style | Underline search, same style in the home overlay and on the results page: a single, steady grey line (no animation), placeholder "Search…", with a small magnifying-glass button on the right (both pages). The line turns a lighter grey while the box is selected (so keyboard users can see where they are). The button just turns white on hover (no backing). While there's text, a small × appears just left of it: it clears the box and keeps the cursor there (Escape does the same). Custom rather than the browser's own clear button, which looks different in every browser. Enter goes to `/search?q=...`. |
| Motion | With "reduce motion" on: still volumes (no bob, no mouse depth, no cover swaps), no entrance animation. The bob and swaps pause while the tab is hidden or the search box is open. |
| Loading / failure | Search (the overlay) works immediately; the volumes fade in once the list arrives, and each cover shows its own colour until its image loads. If the covers can't load, the page simply has no volumes. |

App name: **sankomanga** (the domain will be English too). Tagline: "explore
your next manga". Both live in one place each (`web/static/js/config.js`,
plus `appName` in `config.go` for the backend), as do the part shown in grey
(`GREY_WORD`) and the Japanese name (`JAPANESE_NAME`). The name is written
lowercase in text (tab titles, the domain); the wordmarks, big and small,
are set in capitals (a logo can be styled differently from the name in
text). Logo: the home
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
- Covers as 3D manga books (turned 32°, cream page blocks on the left edge
  and top, back cover), and then with a made-up spine (AniList has no
  spine images): distinctive, but it needed invented parts to look real,
  and it was the only 3D object on the site (replaced by flat covers,
  matching the other pages).
- Volumes as a flat tilted cover with a thin page strip, and covers that
  faded out to blank before the next faded in: looked like a card, not a
  book, and the swap flashed (replaced by full 3D books and a crossfade).
- Two rows of covers gliding endlessly across the top and bottom, the name
  between them (replaced by the floating volumes: calmer, more space, and
  much lighter to load).
- A caption row under the name (search hint left, credit right, lined up
  with the name's edges): moved to the bottom corners with the volumes.
- The hint and the credit as two separate centred lines under the tagline:
  next to the huge name they looked like leftovers (replaced by the caption
  row).
- SANKO and サンコー crossfading as whole words, 4 seconds each: smooth but
  generic; replaced by the sound-by-sound swap.
- サンコー as a huge, very dark watermark behind the name (replaced by
  SANKO and サンコー taking turns in the name itself).
- MANGA as an image with black-and-white manga art inside each letter: the
  only art available showed existing series' characters (fine as a private
  preview, not for a public logo); replaced by MANGA in grey, matching the
  small wordmark, which is simpler, faster and the same mark everywhere.
- A manga page texture drawn in CSS inside MANGA (screentone, speed lines,
  hatching, panel gutters): at this size it read as stripes, like a glitch.

## Results screen

Centred, like the home page, on the same pure black.

| Part | Decision |
|---|---|
| Header | Lined up with the left edge of the grid: the mini wordmark (links home; see "Mini wordmark" below); then a smaller version of the underline search holding the current search term ("/" jumps to it and selects the text) (280px on desktop; full width on phones, lining up with both edges of the grid); then the count in small italic grey: "6 results", or "20 closest matches" when there are more pages (no grand total: AniList reports 5000 for every search). Kept to what the cards can't tell you. |
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
one line above it, lined up with its edges: the mini wordmark on the left
(links home) and "back to [search icon]" on the right. "Back to" goes
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
("Page not found" / "There's nothing at this address." / the mini wordmark, linking home)
with a real 404 status. `/favicon.ico` serves the site's icon.

## Errors

Friendly wording, e.g. "AniList is busy right now, try again in a few seconds."

## Frontend approach

Plain HTML, CSS and JavaScript, no framework and no build step. Each screen
is its own HTML page with one small script; Go serves them together with the
API (so no CORS is needed) and bundles them into the binary with `embed`.
Styles are split the same way: `base.css` holds what every page shares
(tokens, base, footer, search bar, the small wordmark and "back to" links,
the 404 page, touch tap areas), and each page adds its own file
(`home.css`, `results.css`, `title.css`); the 404 page needs only
`base.css`.
Visual design is not limited by this: anything CSS can do is possible, and
libraries can be added for specific effects if ever needed.

Outside resources in use: only the Inter font, from Google Fonts. It could
be self-hosted before deploying, so visitors' browsers don't contact a third
party.

## AniList notes

Moved here from the API contract: internal notes about how the server talks
to AniList. The queries in use today live in `anilist.go`; the ones below may
be older versions.

These are for reference. They're internal and the frontend never sees them.

**Filters on every query** (names confirmed by schema introspection):
`type: MANGA, countryOfOrigin: "JP", format_in: [MANGA, ONE_SHOT], isAdult: false`.
Search also uses `sort: [SEARCH_MATCH]`.

**Decision (2026-10-24):** `isAdult: false` is the only adult filter for now.
Ecchi-genre titles (e.g. My Dress-Up Darling, Fairy Tail, To LOVE-Ru) do appear.
We can revisit this later with `genre_not_in: ["Ecchi"]`.

**Rate limit:** the AniList docs currently say the API is temporarily degraded to
**30 requests/minute** (normally 90), plus an unpublished burst limiter. Going over
gives 429 + `Retry-After` and a 1-minute lockout. Cache and limit choices
(step 7) will be designed around 30/min.

**Search query**
```graphql
query ($search: String, $page: Int) {
  Page(page: $page, perPage: 20) {
    pageInfo { hasNextPage }
    media(search: $search, type: MANGA, countryOfOrigin: "JP",
          format_in: [MANGA, ONE_SHOT], isAdult: false, sort: [SEARCH_MATCH]) {
      id
      title { romaji english native }
      coverImage { large color }
      startDate { year }
      format
      status
      averageScore
    }
  }
}
```

**Detail query**
```graphql
query ($id: Int) {
  Media(id: $id, type: MANGA, countryOfOrigin: "JP",
        format_in: [MANGA, ONE_SHOT], isAdult: false) {
    id
    title { romaji english native }
    coverImage { extraLarge color }
    startDate { year }
    format status chapters volumes averageScore genres description siteUrl
  }
}
```
For an unknown or filtered id this returns HTTP 404 with `{"data":{"Media":null}}`.
We map that to our own 404.

**Showcase query**
```graphql
query ($perPage: Int) {
  Page(page: 1, perPage: $perPage) {
    media(type: MANGA, countryOfOrigin: "JP", format_in: [MANGA, ONE_SHOT],
          isAdult: false, genre_not_in: ["Ecchi"], sort: [POPULARITY_DESC]) {
      id
      title { romaji english native }
      coverImage { extraLarge color }
    }
  }
}
```

**How we map AniList failures**
| AniList | Us |
|---|---|
| 404 on detail | 404 |
| 429 | 503 (+ `Retry-After` passed through) |
| Timeout or can't connect | 503 |
| 5xx, unreadable JSON, or a GraphQL `errors` array | 502, and we log it |

Rule of thumb: **503 = "AniList is busy or down, try again soon"**,
**502 = "AniList answered but the answer was broken"**.
