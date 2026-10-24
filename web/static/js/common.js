// Small helpers shared by every page.

import { APP_NAME, TAGLINE } from "./config.js";

// fillBranding puts the app name and tagline into the page: the browser
// tab title, and any element marked with data-app-name or data-tagline.
// pageTitle is an optional page-specific part, e.g. a manga's title.
export function fillBranding(pageTitle) {
  document.title = pageTitle ? `${pageTitle} · ${APP_NAME}` : `${APP_NAME} · ${TAGLINE}`;

  for (const element of document.querySelectorAll("[data-app-name]")) {
    element.textContent = APP_NAME;
  }
  for (const element of document.querySelectorAll("[data-tagline]")) {
    element.textContent = TAGLINE;
  }
}

// errorMessage turns an API error status code into friendly words, so
// every page describes problems the same way.
export function errorMessage(statusCode) {
  switch (statusCode) {
    case 429:
      return "You're going quite fast. Try again in a few seconds.";
    case 502:
    case 503:
      return "AniList is busy right now. Try again in a few seconds.";
    default:
      return "Something went wrong. Try again in a moment.";
  }
}

// addClearButton adds a small "×" to a search form that empties the search
// box. It only shows while there's text (the CSS shows it when the form has
// the "has-text" class). Clearing keeps the cursor in the box, ready for a
// new search; it doesn't search or reload anything. Escape clears too.
export function addClearButton(form) {
  const input = form.querySelector("input[name='q']");

  const clear = document.createElement("button");
  clear.type = "button"; // "button", so clicking it never submits the form
  clear.className = "search-clear";
  clear.setAttribute("aria-label", "Clear search");
  clear.title = "Clear";
  // A fixed, data-free icon, so innerHTML is safe here.
  clear.innerHTML =
    '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" aria-hidden="true">' +
    '<path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>';
  // Placed before the search button, so the Tab key reaches them in the
  // same order you see them (the × sits to the left of the button).
  form.insertBefore(clear, form.querySelector(".search-button"));

  const showOrHide = () => form.classList.toggle("has-text", input.value !== "");

  const clearText = () => {
    input.value = "";
    showOrHide();
    input.focus();
  };

  clear.addEventListener("click", clearText);
  input.addEventListener("input", showOrHide);
  input.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && input.value !== "") {
      event.preventDefault();
      clearText();
    }
  });
  // When the visitor comes back with the browser's back button, the box
  // may be refilled from before; check again so the × matches.
  window.addEventListener("pageshow", showOrHide);

  showOrHide();
}

// focusSearchOnSlash makes the "/" key jump to a search box from anywhere
// on the page (a common shortcut on sites like GitHub and YouTube), and
// selects its text so you can type a new search straight away. It's
// ignored while you're already typing in a box.
export function focusSearchOnSlash(input) {
  document.addEventListener("keydown", (event) => {
    if (event.key !== "/" || event.ctrlKey || event.metaKey || event.altKey) {
      return;
    }
    const typingSomewhere = event.target.closest("input, textarea, [contenteditable]");
    if (typingSomewhere) {
      return;
    }
    event.preventDefault(); // don't type the "/" itself
    input.focus();
    input.select();
  });
}

// preventEmptySearch stops a search form from submitting when the box only
// contains spaces (the API would answer "query must not be empty"), and
// trims spaces off the ends of the query before it's sent.
export function preventEmptySearch(form) {
  const input = form.querySelector("input[name='q']");
  form.addEventListener("submit", (event) => {
    input.value = input.value.trim();
    if (input.value === "") {
      event.preventDefault();
      input.focus();
    }
  });
}
