# sankomanga

A minimalist, read-only reference site for Japanese manga: search a title,
browse the results, and see the key facts on a title page, with a link out
to AniList. No accounts, no lists, no reading.

サンコー (sankō) is the sound of 参考, "reference".

Data comes from the public [AniList GraphQL API](https://docs.anilist.co).
sankomanga is not affiliated with AniList.

## What's inside

- **Backend:** Go, standard library only (no dependencies). One small server
  that answers the JSON API and serves the website.
- **Frontend:** plain HTML, CSS and JavaScript modules, no framework and no
  build step. The Go server bundles the `web/` folder into its program.

## Running it

You need [Go](https://go.dev/dl/) 1.27 or newer.

```bash
go run .
```

Then open <http://localhost:8080>.

To use a different port, set the `PORT` environment variable (hosting
platforms usually do this for you):

```bash
PORT=3000 go run .
```

Go bundles the `web/` folder into the program when it builds, so **restart
the server after changing any HTML, CSS or JavaScript** to see the change.

To build a single program file you can run anywhere:

```bash
go build -o sankomanga .
```

## Testing

```bash
go test ./...
```

Before committing, it's worth also running the checks below: `gofmt` lists
any file that isn't formatted the Go way (nothing printed means all good),
`go vet` catches common mistakes, and `-race` makes the tests also look for
two parts of the program touching the same data at once.

```bash
gofmt -l . && go vet ./... && go test -race ./...
```

## The API

| Endpoint | What it returns |
|---|---|
| `GET /api/search?q=...&page=1` | Up to 20 manga cards for a search (`page` is optional) |
| `GET /api/manga/{id}` | One title's details (404 for adult, non-Japanese or novel titles) |
| `GET /api/showcase` | The 30 most popular covers, for the home page |
| `GET /api/health` | `{"status":"ok"}` when the server is up |

Errors always look like `{"error": "..."}`: 400 for a bad request, 404 for an
unknown or filtered title, 429 when one visitor asks too fast, 502/503 when
AniList is down or busy. The full contract, with every field and rule, is in
[docs/api-contract.md](docs/api-contract.md).

Every AniList query is limited to Japanese manga and one-shots (no light
novels) and excludes adult titles.

## Where things are

| File | What it does |
|---|---|
| `main.go` | Starts the server, keeps the home covers ready, shuts down cleanly |
| `config.go` | The app's name and the default port |
| `routes.go` | Which address goes to which handler |
| `handlers_public.go` | The API handlers |
| `api_types.go` | The JSON the API sends |
| `display.go` | Turns AniList's raw data into what we show (titles, scores, synopsis cleanup) |
| `anilist.go` | The only file that talks to AniList |
| `cache.go` | Short-lived in-memory cache, with expiry and a size cap |
| `ratelimit.go` | Limits for each visitor, and for our calls to AniList |
| `middleware.go`, `compress.go`, `respond.go` | Logging, gzip, and shared response helpers |
| `web.go` | Serves the pages, CSS, JavaScript and tab icon |
| `web/` | The website: `index.html` (home), `search.html` (results), `manga.html` (title page), `404.html` |
| `web/static/js/config.js` | The frontend's name, tagline and wordmark settings |
| `docs/design.md` | Every design decision, and what was tried and dropped |
| `docs/api-contract.md` | The API contract |

Each `*_test.go` file holds the tests for the file of the same name.

## Changing the name

The name lives in two places: `appName` in `config.go` (backend) and
`APP_NAME` in `web/static/js/config.js` (frontend). Next to `APP_NAME` are
`GREY_WORD` (the part of the name drawn as an outline, "manga") and
`NAME_SOUNDS` (the name's sounds and their katakana, for the home page's
SANKO ↔ サンコー animation).

## Playing fair with AniList

AniList's [terms of use](https://docs.anilist.co/guide/terms-of-use) shape a
few choices here:

- **No storing their data.** The cache lives only in memory, every entry
  expires (10 minutes for searches, 1 hour for title pages, 6 hours for the
  home covers), and it holds at most 1,000 entries.
- **No mass collection.** Data is only fetched when someone looks something
  up, plus the home covers about five times a day.
- **Their rate limit** (currently 30 requests a minute): we stay at 20 a
  minute in total and pause when AniList says "too many requests". This limit
  is counted inside the server, so **run a single copy of it** when hosting.
- **Credit:** every page says "Data from AniList", and each title page links
  to it on AniList.
- **No list or progress tracking:** AniList doesn't allow competing tracker
  services.
