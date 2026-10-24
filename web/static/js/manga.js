// The title page (/manga/{id}): reads the id from the page's address, asks
// our API for that manga, and fills in the cover, title, facts, score,
// genres, synopsis and AniList link. Also handles loading, "not found"
// and errors.
//
// Every piece of text from the API is put on the page with textContent,
// never as raw HTML, so an unusual title or synopsis can never inject code.

import { errorMessage, fillBranding } from "./common.js";

const article = document.querySelector(".title");
const message = document.querySelector(".title-message");

fillBranding();
setUpBackToResults();

// The address looks like /manga/53390; the id is the last part.
const id = window.location.pathname.split("/").pop();

if (/^\d+$/.test(id)) {
  loadTitle(id);
} else {
  showNotFound(); // not a number, so it can't be a manga
}

// loadTitle asks the API for one manga and shows whatever comes back.
async function loadTitle(id) {
  let response;
  try {
    response = await fetch(`/api/manga/${id}`);
  } catch {
    // The request never got an answer (e.g. the visitor is offline).
    showMessage("Something went wrong. Try again in a moment.");
    return;
  }

  // 404: unknown, or filtered out (adult, not Japanese, a novel).
  // 400: an id the API can't accept (e.g. far too large).
  if (response.status === 404 || response.status === 400) {
    showNotFound();
    return;
  }
  if (!response.ok) {
    showMessage(errorMessage(response.status));
    return;
  }

  renderTitle(await response.json());
}

// --- Showing the title --------------------------------------------------

// renderTitle fills in every part of the page from the API's answer.
// Parts that are missing (no subtitle, no score...) are left empty, and
// the CSS hides empty parts, so nothing ever shows as blank or "—".
function renderTitle(manga) {
  fillBranding(manga.title); // browser tab: "Shingeki no Kyojin · manga-app"

  renderCover(manga);
  article.querySelector(".title-name").textContent = manga.title;
  article.querySelector(".title-subtitle").textContent = manga.subtitle ?? "";
  article.querySelector(".title-facts").textContent = factsLine(manga);
  article.querySelector(".title-score").textContent =
    manga.score === null ? "" : `★ ${manga.score.toFixed(1)}`; // 8 → "8.0"

  const genres = article.querySelector(".title-genres");
  for (const genre of manga.genres) {
    const item = document.createElement("li");
    item.textContent = genre;
    genres.append(item);
  }

  const synopsis = article.querySelector(".title-synopsis");
  for (const paragraph of manga.synopsis) {
    const text = document.createElement("p");
    text.textContent = paragraph;
    synopsis.append(text);
  }

  article.querySelector(".title-link").href = manga.url;

  article.classList.remove("is-loading");
  article.removeAttribute("aria-busy");
}

// renderCover adds the large cover image. Until it has downloaded, the
// cover area shows the cover's own colour (dimmed); then the image fades in.
function renderCover({ title, coverUrl, coverColor }) {
  const cover = article.querySelector(".title-cover");
  if (coverColor) {
    cover.style.setProperty("--cover-color", coverColor);
  }
  if (!coverUrl) {
    return;
  }

  const image = document.createElement("img");
  image.alt = `Cover of ${title}`;
  image.decoding = "async";
  image.addEventListener("load", () => image.classList.add("is-loaded"), { once: true });
  image.src = coverUrl;
  cover.append(image);
}

// factsLine builds e.g. "Manga · Finished · 2009 · 141 chapters · 34 volumes",
// leaving out anything missing. (Ongoing series often have no counts.)
//
// Non-breaking spaces (\u00a0) keep each part whole ("141 chapters") and
// keep each "·" with the part before it, so the line only breaks between
// parts on narrow screens.
function factsLine({ format, status, year, chapters, volumes }) {
  const parts = [format];
  if (status !== null) {
    parts.push(status);
  }
  if (year !== null) {
    parts.push(String(year));
  }
  if (chapters !== null) {
    parts.push(chapters === 1 ? "1 chapter" : `${chapters} chapters`);
  }
  if (volumes !== null) {
    parts.push(volumes === 1 ? "1 volume" : `${volumes} volumes`);
  }
  return parts.map((part) => part.replaceAll(" ", "\u00a0")).join("\u00a0· ");
}

// --- Back to the search results -----------------------------------------

// setUpBackToResults makes the search icon lead back to the same search
// results (never a fresh, empty search):
//  - If the visitor came here from our results page, it simply goes
//    "back", which also keeps their scroll position in the results.
//  - Otherwise (e.g. a shared link), it opens the results for the search
//    carried in this page's address (?q=berserk, added by the results page).
//  - With no search at all, it goes to the home page.
function setUpBackToResults() {
  const link = document.querySelector(".back-to-results");
  const query = new URLSearchParams(window.location.search).get("q")?.trim();

  link.href = query ? `/search?q=${encodeURIComponent(query)}` : "/";

  link.addEventListener("click", (event) => {
    if (cameFromResultsPage()) {
      event.preventDefault();
      history.back();
    }
  });
}

// cameFromResultsPage reports whether the previous page was our own
// results page (same site, path /search).
function cameFromResultsPage() {
  if (!document.referrer) {
    return false;
  }
  const previous = new URL(document.referrer);
  return previous.origin === window.location.origin && previous.pathname === "/search";
}

// --- Not found and errors -----------------------------------------------

// showNotFound replaces the title with a short message and a way home.
function showNotFound() {
  showMessage("This title isn't available here. ");

  const home = document.createElement("a");
  home.href = "/";
  home.textContent = "Search for another";
  message.append(home);
}

// showMessage hides the title area and shows one calm line instead.
function showMessage(text) {
  article.hidden = true;
  message.textContent = text;
}
