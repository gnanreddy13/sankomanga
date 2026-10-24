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
