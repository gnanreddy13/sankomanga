// The home page's search: instead of a search bar on the page, the search
// box opens on top of the blurred page ("spotlight" style).
//
// It opens by:
//   - just starting to type (the first letter goes straight into the box)
//   - pressing "/"
//   - clicking or tapping anywhere on the page
//   - the "Type to search" / "Tap anywhere to search" hint (a real button,
//     for keyboard and screen-reader users)
// It closes with Esc, a click or tap outside the box, or the browser's
// back button / back gesture.

import { addClearButton, preventEmptySearch } from "./common.js";

// How long the closing fade takes (the same as .is-closing in home.css).
const CLOSE_MS = 200;

const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

export function setUpSearchOverlay() {
  const overlay = document.querySelector(".search-overlay");
  const form = overlay.querySelector(".search");
  const input = overlay.querySelector("input[name='q']");
  const hint = document.querySelector(".search-hint");

  preventEmptySearch(form);
  addClearButton(form);

  // Searching from the overlay: replace the overlay's extra history step
  // (see open) with the results page, instead of adding one more. Then
  // "back" on the results page goes straight to the home page, rather
  // than to the home page with the search still "open".
  form.addEventListener("submit", (event) => {
    if (event.defaultPrevented || !history.state?.searchOpen) {
      return; // an empty search (blocked), or no extra step to replace
    }
    event.preventDefault();
    const query = new URLSearchParams(new FormData(form)).toString(); // "q=one+piece"
    window.location.replace(`${form.action}?${query}`);
  });

  // Where the keyboard focus was before opening, to put it back on close.
  let focusBeforeOpening = null;

  // While closing, the overlay is still on screen (fading out) but counts
  // as closed: typing or clicking can open it again straight away.
  let closingTimer = null;

  const isOpen = () => !overlay.hidden && closingTimer === null;

  // open shows the search box, optionally starting with some text (the
  // letter that was typed to open it), and puts the cursor in it.
  function open(startText = "") {
    if (isOpen()) {
      return;
    }
    stopClosing(); // opened again while it was still fading out
    focusBeforeOpening = document.activeElement;
    overlay.hidden = false;
    document.body.classList.add("search-open");

    input.value = startText;
    // Let addClearButton know the text changed, so the × shows if needed.
    input.dispatchEvent(new Event("input"));
    input.focus();

    // An extra history step, so the browser's back button (or the phone's
    // back gesture) closes the search instead of leaving the site.
    history.pushState({ searchOpen: true }, "");
  }

  // close hides the search box. If we added a history step for it, going
  // back removes that step; the "popstate" handler below then hides it.
  function close() {
    if (!isOpen()) {
      return;
    }
    if (history.state?.searchOpen) {
      history.back();
    } else {
      hide();
    }
  }

  // hide fades the overlay out (see .is-closing in home.css), then hides
  // it for good. The page is usable straight away; only the picture of
  // the overlay lingers for the fade. No fade with "reduce motion".
  function hide() {
    document.body.classList.remove("search-open");
    focusBeforeOpening?.focus?.();

    if (reduceMotion) {
      overlay.hidden = true;
      return;
    }
    overlay.classList.add("is-closing");
    closingTimer = setTimeout(() => {
      overlay.hidden = true;
      stopClosing();
    }, CLOSE_MS);
  }

  // stopClosing ends a fade-out early (or tidies up after one).
  function stopClosing() {
    clearTimeout(closingTimer);
    closingTimer = null;
    overlay.classList.remove("is-closing");
  }

  // The browser's back button / back gesture while the search is open.
  window.addEventListener("popstate", () => {
    if (isOpen()) {
      hide();
    }
  });

  // --- Ways to open -----------------------------------------------------

  hint.addEventListener("click", (event) => {
    event.stopPropagation(); // the "click anywhere" handler below would open it twice
    open();
  });

  // Click or tap anywhere on the page opens it, except on links (e.g. the
  // AniList logo in the footer) and while it's already open.
  document.addEventListener("click", (event) => {
    if (isOpen() || event.target.closest("a")) {
      return;
    }
    open();
  });

  document.addEventListener("keydown", (event) => {
    if (isOpen()) {
      handleKeyWhileOpen(event);
      return;
    }
    // Leave shortcuts (Ctrl/⌘/Alt + key) to the browser.
    if (event.ctrlKey || event.metaKey || event.altKey) {
      return;
    }
    if (event.key === "/") {
      event.preventDefault(); // don't type the "/" itself
      open();
    } else if (isTypedCharacter(event.key)) {
      event.preventDefault();
      open(event.key); // the letter becomes the start of the search
    }
  });

  // --- While open -------------------------------------------------------

  // A click or tap on the blurred area around the box closes it.
  overlay.addEventListener("click", (event) => {
    event.stopPropagation(); // don't let the page's "click anywhere" reopen it
    if (event.target === overlay) {
      close();
    }
  });

  function handleKeyWhileOpen(event) {
    if (event.key === "Escape") {
      // The first Esc clears any text (addClearButton handles that and
      // marks the event as handled); Esc on an empty box closes it.
      if (!event.defaultPrevented) {
        event.preventDefault();
        close();
      }
      return;
    }
    if (event.key === "Tab") {
      keepFocusInside(event);
    }
  }

  // keepFocusInside makes Tab cycle through the box's controls only (the
  // input, the × and the search button), not the page hidden behind.
  function keepFocusInside(event) {
    const controls = [...overlay.querySelectorAll("input, button")].filter(
      (element) => element.offsetParent !== null, // only visible ones (the × may be hidden)
    );
    const first = controls[0];
    const last = controls[controls.length - 1];

    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }
}

// isTypedCharacter reports whether a key press types a visible character
// (a letter, digit, kana...), as opposed to keys like Shift, Enter, arrows
// or Tab. Space is left out: it shouldn't open the search by itself.
function isTypedCharacter(key) {
  return key.length === 1 && key !== " ";
}
