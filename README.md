# sankomanga

[![Tests](https://github.com/gnanreddy13/sankomanga/actions/workflows/test.yml/badge.svg)](https://github.com/gnanreddy13/sankomanga/actions/workflows/test.yml)

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

## Deploying (Railway)

sankomanga is one Go program with the website built in, so any host that
can run Go works. These steps are for [Railway](https://railway.com).

1. **Put the code on GitHub.** A private repository is fine.
2. **Create the service.** On Railway: New Project → Deploy from GitHub
   repo → pick the repository. Railway sees `go.mod`, builds the program
   and starts it. It also sets `PORT` itself, which the app already reads.
3. **Set two variables** (the service's Variables tab):

   | Variable | Value | Why |
   |---|---|---|
   | `TRUSTED_PROXY_HEADER` | `X-Real-IP` | Railway's proxy passes each visitor's address in this header. Without it, every visitor looks like the proxy and they'd all share one rate limit. |
   | `SITE_URL` | `https://your-domain` | Only with your own domain. Without it, link previews use Railway's address automatically (`RAILWAY_PUBLIC_DOMAIN`). |

4. **Check two settings** (the service's Settings tab):
   - **Replicas: 1.** The AniList rate limit is counted inside the program, so
     there must be exactly one copy. (1 is Railway's default.)
   - **Healthcheck path: `/api/health`.** Railway then only switches to a new
     version once it answers, so a broken deploy never replaces a working one.
5. **Get an address.** Settings → Networking → Generate Domain (or add your
   own domain there, then set `SITE_URL`).

After that, every push to GitHub deploys a new version.

**After the first deploy, check:**

- `https://your-address/api/health` answers `{"status":"ok"}`.
- The home page, a search and a title page work.
- The rate limit tells visitors apart: from one computer, about 40 quick
  searches in a row should get "You're going quite fast", while a phone on
  mobile data can still search normally at the same moment.
- A link preview shows the wordmark: paste the address into a chat app.

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

GitHub runs these same checks on every push
([.github/workflows/test.yml](.github/workflows/test.yml)) and shows a ✓ or
✗ next to each commit; the badge at the top shows the latest result.

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
| `config.go` | The app's name, the port, the site's public address, and the proxy header for visitors' addresses |
| `routes.go` | Which address goes to which handler |
| `handlers_public.go` | The API handlers |
| `api_types.go` | The JSON the API sends |
| `display.go` | Turns AniList's raw data into what we show (titles, scores, synopsis cleanup) |
| `anilist.go` | The only file that talks to AniList |
| `cache.go` | Short-lived in-memory cache, with expiry and a size cap |
| `ratelimit.go` | Limits for each visitor, and for our calls to AniList |
| `middleware.go`, `compress.go`, `respond.go` | Logging, gzip, and shared response helpers |
| `web.go` | Serves the pages (filling in the name and address for link previews), CSS, JavaScript and images, with ETags so repeat visits are cheap |
| `web/` | The website: `index.html` (home), `search.html` (results), `manga.html` (title page), `404.html`, plus the tab icon, the iPhone home-screen icon and the link preview picture |
| `web/static/css/` | `base.css` (what every page shares), plus one file per page: `home.css`, `results.css`, `title.css` |
| `web/static/js/` | One script per page (`home.js`, `search.js`, `manga.js`), plus `wordmark.js` (the home page name and its サンコー animation), `search-overlay.js` (the home search box) and `common.js` (shared helpers) |
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
