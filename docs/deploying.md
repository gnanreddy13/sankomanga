# Deploying

sankomanga is one Go program with the website built in, so any host that can
run Go works. These steps are for [Railway](https://railway.com).

## Steps

1. **Create the service.** New Project → Deploy from GitHub repo → pick the
   repository. Railway sees `go.mod`, builds the program and starts it, and
   sets `PORT` itself.
2. **Set the variables** (Variables tab):

   | Variable | Value | Why |
   |---|---|---|
   | `TRUSTED_PROXY_HEADER` | `X-Real-IP` | Railway's proxy passes each visitor's address in this header. Without it, all visitors share one rate limit. |
   | `SITE_URL` | `https://your-domain` | Only with your own domain. Otherwise link previews use Railway's address automatically. |

3. **Check two settings** (Settings tab):
   - **Replicas: 1** (the default). The AniList rate limit is counted inside
     the program, so there must be exactly one copy.
   - **Healthcheck path: `/api/health`**, so a broken deploy never replaces a
     working one.
4. **Get an address:** Settings → Networking → Generate Domain, or add your
   own domain and set `SITE_URL`.

Every push to GitHub then deploys a new version.

## After the first deploy

- `/api/health` answers `{"status":"ok"}`.
- The home page, a search and a title page work.
- Visitors are told apart: about 40 quick searches from one computer should
  get "You're going quite fast", while a phone on mobile data still searches
  normally.
- Pasting the address into a chat app shows the link preview.

## Playing fair with AniList

AniList's [terms of use](https://docs.anilist.co/guide/terms-of-use) shape a
few choices:

- **No storing their data:** the cache lives only in memory, every entry
  expires (10 minutes for searches, 1 hour for title pages, 6 hours for the
  home covers), and it holds at most 1,000 entries.
- **No mass collection:** data is fetched only when someone looks something
  up, plus the home covers about five times a day.
- **Their rate limit** (currently 30 a minute): the app stays at 20 a minute
  and pauses when AniList says "too many requests". Hence one copy only.
- **Credit:** every page says "Data from AniList", and each title page links
  to it.
- **No list or progress tracking:** AniList doesn't allow competing tracker
  services.
