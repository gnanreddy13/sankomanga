# API contract (draft for review)

The Go backend's public JSON API. The frontend only talks to these endpoints;
it never calls AniList directly.

## General rules

- All endpoints are `GET` and return JSON (`Content-Type: application/json; charset=utf-8`).
- Field names are camelCase.
- **Nullable means the key is always present, with value `null`.** The frontend can
  check `x === null` and never has to wonder whether a key is missing.
- Lists are never `null`: an empty list is `[]`.
- Every response has a `Server-Timing` header with the backend time and whether
  the answer came from the cache, e.g. `Server-Timing: app;dur=12.4, cache;desc="hit"`.
  (Added in build step 7.)
- Any other method (POST, etc.) gets `405` with the standard error body.

## Errors

Every error uses the same shape:

```json
{ "error": "human-readable message" }
```

| Status | When | Example message |
|---|---|---|
| 400 | Empty/too-long query, bad `page`, non-numeric id | `"query must not be empty"` |
| 404 | Unknown id, or a title our filters exclude (adult, non-Japanese, novel, anime) | `"manga not found"` |
| 405 | Method other than GET | `"method not allowed"` |
| 429 | This client is sending us too many requests (our per-IP limit, step 7) | `"too many requests, try again shortly"` |
| 500 | Bug on our side | `"internal error"` |
| 502 | AniList answered with something we couldn't use (5xx, bad JSON, GraphQL error) | `"upstream service error"` |
| 503 | AniList is rate-limiting us (429), timed out, or unreachable | `"upstream service unavailable, try again shortly"` |

A `429` or `503` includes a `Retry-After` header (seconds) when we know it.

A search with no matches is **not** an error: it's a `200` with `"results": []`.
The frontend shows the friendly empty state for that.

---

## `GET /api/health`

Used to check that the server is up. Doesn't call AniList.

```json
{ "status": "ok" }
```

---

## `GET /api/search?q=...&page=1`

### Parameters

| Name | Required | Rules |
|---|---|---|
| `q` | yes | Leading and trailing spaces are trimmed. Must be 1–100 characters after trimming, otherwise `400`. |
| `page` | no | Defaults to `1`. Must be a whole number from 1 to 50, otherwise `400`. |

The 50-page cap (1,000 results) is there to stop anyone crawling AniList's
catalog through us. Real users never page that far.

### Response `200`

```json
{
  "query": "one piece",
  "page": 1,
  "hasNextPage": true,
  "results": [
    {
      "id": 30013,
      "title": "ONE PIECE",
      "coverUrl": "https://s4.anilist.co/file/anilistcdn/media/manga/cover/medium/bx30013-BeslEMqiPhlk.jpg",
      "coverColor": "#f1935d",
      "year": 1997,
      "format": "Manga",
      "status": "Ongoing",
      "score": 9.1
    }
  ]
}
```

- `query` is the trimmed query, so the frontend can show "Results for …".
- `hasNextPage` tells the frontend whether there are more results than this page
  (it shows "20 closest matches" then). It also lets "load more" be added later.
- There is no total count: AniList reports 5000 for every search (checked: even
  "berserk", which has 6 results), so the number would be wrong.
- Results are AniList's best-match order. At most 20 per page.

### Card fields

| Field | Type | Nullable | Source / rule |
|---|---|---|---|
| `id` | int | no | `id` |
| `title` | string | no | See [Title rules](#title-rules) |
| `coverUrl` | string | yes | `coverImage.large` (small enough for cards) |
| `coverColor` | string | yes | `coverImage.color`, e.g. `"#f1935d"`. Placeholder colour while the image loads. |
| `year` | int | yes | `startDate.year` |
| `format` | string | no | `"Manga"` or `"One Shot"` |
| `status` | string | yes | See [Status](#status). Added so result cards can mark ongoing titles ("Ongoing"). |
| `score` | number | yes | See [Score](#score) |

---

## `GET /api/manga/{id}`

### Parameters

`{id}` must be a positive whole number, otherwise `400 "invalid id"`.
An id that AniList doesn't know, or that our filters exclude, gives `404`.
(AniList itself answers 404 in both cases, so they share one code path.)

### Response `200`

```json
{
  "id": 53390,
  "title": "Shingeki no Kyojin",
  "subtitle": "Attack on Titan",
  "format": "Manga",
  "status": "Finished",
  "year": 2009,
  "chapters": 141,
  "volumes": 34,
  "score": 8.4,
  "genres": ["Action", "Drama", "Fantasy", "Mystery"],
  "synopsis": [
    "In this post-apocalyptic sci-fi story, humanity has been devastated by ...",
    "Note: Includes two extra chapters."
  ],
  "coverUrl": "https://s4.anilist.co/file/anilistcdn/media/manga/cover/large/bx53390-1RsuABC34P9D.jpg",
  "coverColor": "#d6431a",
  "url": "https://anilist.co/manga/53390"
}
```

### Detail fields

| Field | Type | Nullable | Source / rule |
|---|---|---|---|
| `id` | int | no | `id` |
| `title` | string | no | See [Title rules](#title-rules) |
| `subtitle` | string | yes | See [Title rules](#title-rules) |
| `format` | string | no | `"Manga"` or `"One Shot"` |
| `status` | string | yes | See [Status](#status) |
| `year` | int | yes | `startDate.year` |
| `chapters` | int | yes | `chapters`. Usually `null` for ongoing series. |
| `volumes` | int | yes | `volumes`. Usually `null` for ongoing series. |
| `score` | number | yes | See [Score](#score) |
| `genres` | string[] | no (may be `[]`) | `genres` |
| `synopsis` | string[] | no (may be `[]`) | `description`, cleaned. See [Synopsis](#synopsis). |
| `coverUrl` | string | yes | `coverImage.extraLarge` |
| `coverColor` | string | yes | `coverImage.color` |
| `url` | string | no | `siteUrl`, used for the "View on AniList" link |

---

## `GET /api/showcase`

Covers for the decorative animation on the home page: the most popular
Japanese manga, **with the Ecchi genre excluded** (for this endpoint only;
search is unchanged). Added after the original contract, for the home
screen design.

### Response `200`

```json
{
  "covers": [
    {
      "id": 105778,
      "title": "Chainsaw Man",
      "coverUrl": "https://s4.anilist.co/file/anilistcdn/media/manga/cover/medium/....jpg",
      "coverUrl2x": "https://s4.anilist.co/file/anilistcdn/media/manga/cover/large/....jpg",
      "coverColor": "#f1c90d"
    }
  ]
}
```

| Field | Type | Nullable | Source / rule |
|---|---|---|---|
| `id` | int | no | `id` (unused today; keeps clickable covers possible) |
| `title` | string | no | See [Title rules](#title-rules) (unused today) |
| `coverUrl` | string | no | `coverImage.large` (230px wide), for ordinary screens. Titles without a cover are left out. |
| `coverUrl2x` | string | no | `coverImage.extraLarge` (460px wide), for sharp "Retina" screens; the same as `coverUrl` if AniList has no extra-large image. The browser picks between the two. |
| `coverColor` | string | yes | `coverImage.color`, placeholder while the image loads |

- Up to 30 covers, most popular first. `covers` may be `[]`; the frontend
  then leaves the rows empty.
- Cached on the server for 6 hours, and kept ready in the background: the server fetches it on startup and refreshes it every 5 hours (retrying after 2 minutes if AniList is down), so visitors never wait for AniList. `Cache-Control: public, max-age=3600`.
- Counts towards the per-IP limit like the other data endpoints.

---

## Display rules (done in Go)

### Title rules
- **title**: the first non-empty value of romaji, then English, then native,
  with whitespace trimmed. AniList sometimes sends stray whitespace, e.g. `"Attack on Titan Volume 0\r\n"`.
- **subtitle**: the English title if it's non-empty and differs from `title`
  (ignoring upper/lower case). Otherwise the native title if it's non-empty and differs from `title`.
  Otherwise `null`.
  - Example: `ONE PIECE` / `One Piece` / `ONE PIECE`. The English title matches when case is ignored,
    and the native title matches too, so the subtitle is `null`.
  - Example: `Shingeki no Kyojin` / `Attack on Titan`, so the subtitle is `"Attack on Titan"`.
  - Example: `Oyome ni Oideyo!` with no English title, so the subtitle is the native `"お嫁においでよ！"`.
- Both rules live in one Go function so they're easy to change.

### Score
AniList gives `averageScore` as 0–100. We return it divided by 10, rounded to one
decimal place: `84` becomes `8.4`. JSON shows `80` as `8`, so the frontend
formats it with one decimal ("8.0"). `null` stays `null`.

### Status
| AniList | We return |
|---|---|
| `FINISHED` | `"Finished"` |
| `RELEASING` | `"Ongoing"` (clearer than AniList's "Releasing") |
| `NOT_YET_RELEASED` | `"Not yet released"` |
| `CANCELLED` | `"Cancelled"` |
| `HIATUS` | `"On hiatus"` |
| anything else / null | `null` (and we log it) |

### Format
`MANGA` becomes `"Manga"` and `ONE_SHOT` becomes `"One Shot"`. Other formats are already
excluded by the AniList query.

### Synopsis
The HTML `description` becomes an array of plain-text paragraphs:
1. Normalise line endings (`\r\n` becomes `\n`).
2. Split on every `<br>` (also `<br/>`, `<br />`). Each piece is a candidate paragraph.
3. Remove all other tags but keep their text (`<i>`, `</i>`, `<a href=…>`, and unclosed tags too).
4. Decode HTML entities (`&hearts;` becomes `♥`, `&amp;` becomes `&`, `&quot;` becomes `"`).
5. Trim whitespace. Drop empty pieces.
6. Drop any piece that starts with `(Source:` and ends with `)`. Keep everything around it,
   including notes that come after it.
7. A `null` description gives `[]`.

Note lists like Berserk's become one paragraph per bullet
(`"Notes:"`, `"- Volumes 1-5 contain…"`, …). That reads fine as plain text.
