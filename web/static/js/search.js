// The results page (/search?q=...): reads the search term from the page's
// address, asks our API for matching manga, and shows them as a grid of
// cards. Also handles the in-between moments: loading, no results, more
// than one page of results, and errors.
//
// Every piece of text from the API or the address bar is put on the page
// with textContent, never as raw HTML, so an unusual title or search term
// can never inject code. (The only raw HTML is the fixed, data-free markup
// of the loading placeholders.)

import { addClearButton, errorMessage, fillBranding, focusSearchOnSlash, preventEmptySearch } from "./common.js";

// How many placeholder cards to show while loading (one full page).
const PLACEHOLDER_COUNT = 20;

const form = document.querySelector(".search");
const input = document.querySelector("#search-input");
const status = document.querySelector(".results-status");
const grid = document.querySelector(".results-grid");
const note = document.querySelector(".results-note");

const query = new URLSearchParams(window.location.search).get("q")?.trim() ?? "";

if (query === "") {
  // Nothing to search for: the home page is the place to start.
  window.location.replace("/");
} else {
  fillBranding(query);
  input.value = query;
  preventEmptySearch(form);
  addClearButton(form); // after filling in the query, so the × shows right away
  focusSearchOnSlash(input);
  search(query);
}

// search asks the API for results and shows whatever comes back.
async function search(query) {
  showPlaceholders();

  let response;
  try {
    response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
  } catch {
    // The request never got an answer (e.g. the visitor is offline).
    showError("Something went wrong. Try again in a moment.");
    return;
  }

  if (!response.ok) {
    showError(errorMessage(response.status));
    return;
  }

  const data = await response.json();
  renderResults(data);
}

// --- Showing results ----------------------------------------------------

// renderResults fills the grid with one card per result, and sets the
// count above and the note below.
function renderResults({ results, hasNextPage }) {
  grid.replaceChildren(); // remove the placeholders
  grid.removeAttribute("aria-busy");

  if (results.length === 0) {
    status.textContent = `No manga found for “${query}”. Check the spelling, or try the English or Japanese title.`;
    return;
  }

  status.textContent = summaryLine(results, hasNextPage);
  for (const manga of results) {
    grid.append(createCard(manga));
  }

  // Below the grid, only when results are cut off.
  note.textContent = hasNextPage
    ? `Only the ${results.length} closest matches are shown. Try a more specific title.`
    : "";
}

// createCard builds one result: a link to the title page holding the
// cover, the title, and a small info line.
function createCard(manga) {
  const item = document.createElement("li");

  const link = document.createElement("a");
  link.className = "card";
  // The search goes along in the link (/manga/53390?q=berserk), so the
  // title page's search icon can lead back to these same results.
  link.href = `/manga/${manga.id}?q=${encodeURIComponent(query)}`;
  if (manga.coverColor) {
    // Used for the placeholder colour while the image loads.
    link.style.setProperty("--card-color", manga.coverColor);
  }

  const cover = document.createElement("div");
  cover.className = "card-cover";
  if (manga.coverUrl) {
    const image = document.createElement("img");
    image.alt = ""; // the title right below says what it is
    image.loading = "lazy"; // only download covers as they scroll into view
    image.decoding = "async";
    image.addEventListener("load", () => image.classList.add("is-loaded"), { once: true });
    image.src = manga.coverUrl;
    cover.append(image);
  }

  const title = document.createElement("p");
  title.className = "card-title";
  title.textContent = manga.title;

  const meta = document.createElement("p");
  meta.className = "card-meta";
  meta.textContent = infoLine(manga);

  link.append(cover, title, meta);
  item.append(link);
  return item;
}

// infoLine builds e.g. "2009 · Manga · Ongoing · ★ 8.4", leaving out any
// part that's missing (year and score are often unknown). "Ongoing" only
// appears for titles still being published; finished ones stay quiet.
//
// On narrow phone cards the line may need to wrap. Non-breaking spaces
// (\u00a0) keep each part whole ("One Shot", "★ 8.4") and keep each "·"
// with the part before it, so the line only ever breaks between parts.
function infoLine({ year, format, status, score }) {
  const parts = [];
  if (year !== null) {
    parts.push(String(year));
  }
  parts.push(format);
  if (status === "Ongoing") {
    parts.push("Ongoing");
  }
  if (score !== null) {
    parts.push(`★ ${score.toFixed(1)}`); // 8 → "8.0"
  }
  return parts.map((part) => part.replaceAll(" ", "\u00a0")).join("\u00a0· ");
}

// summaryLine says only what the cards can't: how many results there are,
// and whether there are more than shown. e.g. "6 results", or
// "20 closest matches" when there are more pages. (There's no grand total:
// AniList reports 5000 for every search, so it can't be trusted.)
function summaryLine(results, hasMore) {
  const count = results.length;
  if (hasMore) {
    return `${count} closest matches`;
  }
  return count === 1 ? "1 result" : `${count} results`;
}

// --- Loading and errors -------------------------------------------------

// showPlaceholders fills the grid with dark, gently shimmering cards while
// the results are on their way.
function showPlaceholders() {
  status.textContent = "";
  note.textContent = "";
  grid.setAttribute("aria-busy", "true"); // tells screen readers it's loading
  grid.replaceChildren();

  for (let i = 0; i < PLACEHOLDER_COUNT; i++) {
    const item = document.createElement("li");
    item.className = "card card--placeholder";
    item.setAttribute("aria-hidden", "true");
    item.innerHTML = '<div class="card-cover"></div><div class="card-line"></div><div class="card-line card-line--short"></div>';
    grid.append(item);
  }
}

// showError replaces the grid with one calm line of text.
function showError(message) {
  grid.replaceChildren();
  grid.removeAttribute("aria-busy");
  status.textContent = message;
}
