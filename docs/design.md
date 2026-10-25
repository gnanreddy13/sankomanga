# Design

How sankomanga looks and why. The full log of decisions, including
everything tried and dropped, is in [design-history.md](design-history.md).

## Principles

- **Minimal and quiet.** Pure black, white text, one faint grey. The covers
  bring the colour.
- **Only what's useful.** Each screen shows just what you need, and missing
  facts are left out rather than shown as "—".
- **For everyone.** Works with the keyboard, screen readers and touch, and
  every animation stops with "reduce motion".

## Throughout

- **Colours:** black `#000` background. The faintest grey, `#77777f`, still
  passes contrast for small text.
- **Font:** Inter everywhere; Noto Sans JP for サンコー.
- **Wordmark:** **SANKO** in solid white, **MANGA** as a white outline. Big on
  the home page, small (linking home) at the top of the other pages, where
  MANGA glows on hover.
- **Glow:** covers have a soft white glow, the same on every page.
- **Page changes:** a 0.25s crossfade between pages.
- **Credit:** "Data from [AniList logo]" on every page.
- **Icons:** サ on black for the browser tab and the iPhone home screen. Link
  previews show the wordmark on black.

## Home

- **Middle:** the wordmark and the tagline "explore your next manga". On
  phones the name stacks, SANKO over MANGA.
- **SANKO ↔ サンコー:** every 6 seconds SANKO turns into its Japanese spelling,
  sound by sound (SA→サ, N→ン, KO→コー), for 3 seconds. サンコー (sankō)
  sounds like 参考, "reference".
- **Covers:** 8 popular manga float around the name (4 on phones), bobbing
  slowly and shifting a little with the mouse. Every 5 seconds one dissolves
  into another. Decoration only.
- **Corners:** "Type to search" (or "Tap to search") bottom-left, the AniList
  credit bottom-right.
- **Search:** no search bar. Typing, "/", or clicking anywhere opens a search
  box over the blurred page; Esc, clicking outside or "back" closes it.

## Results

- **Header:** the small wordmark, the search line with the current search,
  and the count ("6 results", or "20 closest matches" when there are more).
- **Grid:** 5 covers a row on laptops, 3 on phones. Each card: cover, title,
  and one dim line like `2009 · Manga · Ongoing · ★ 8.4`.
- **Hover:** the cover lifts slightly and glows.
- **Empty or failing:** one calm line, e.g. "No manga found for "xyz"".

## Title page

- **Top:** the small wordmark on the left, "back to [search]" on the right
  (back to the same results).
- **Layout:** the cover on the left, the details on the right (stacked on
  phones).
- **Details:** title and subtitle, one facts line (`Manga · Finished · 2009 ·
  141 chapters · 34 volumes`), the score, genres, the synopsis, and "View on
  AniList ↗".

## 404

A small centred "Page not found" with the small wordmark to go home.

## How it's built

Plain HTML, CSS and JavaScript, no framework and no build step. Each page has
its own HTML file, script and stylesheet (`home.css`, `results.css`,
`title.css`), on top of a shared `base.css`. The Go server serves them and
bundles them into the program.
