// The home page: fills in the name and tagline, fills two rows of popular
// covers (from /api/showcase) that glide slowly across the top and bottom
// of the screen in opposite directions, and sets up the search box that
// opens on top of the page (search-overlay.js).
//
// The gliding itself is CSS (.cover-track in style.css); this file only
// builds the rows and sets their speed.

import { fillBranding } from "./common.js";
import { setUpSearchOverlay } from "./search-overlay.js";

// How fast the rows glide, in pixels per second: slow enough to read as
// calm background movement rather than something to follow.
const GLIDE_PX_PER_SECOND = 22;

const home = document.querySelector(".home");
const topRow = document.querySelector(".cover-row--top");
const bottomRow = document.querySelector(".cover-row--bottom");

fillBranding();
setUpSearchOverlay();
loadRows();
pauseWhileHidden();

// loadRows fetches the covers and fills both rows. The search works the
// whole time; the rows fade in once they're ready. If anything goes wrong,
// the rows simply stay empty (only decoration is missing).
async function loadRows() {
  try {
    const response = await fetch("/api/showcase");
    if (!response.ok) {
      throw new Error(`showcase request failed with status ${response.status}`);
    }
    const { covers } = await response.json();
    if (covers.length === 0) {
      throw new Error("showcase has no covers");
    }

    // Both rows show the same covers, but the bottom one starts halfway
    // through the list, so the same cover is never stacked above itself.
    const half = Math.floor(covers.length / 2);
    fillRow(topRow, covers);
    fillRow(bottomRow, [...covers.slice(half), ...covers.slice(0, half)]);
  } catch (error) {
    console.warn("Cover rows unavailable:", error);
  }
}

// fillRow puts the covers into a row's strip, then a second identical
// copy right after them. The CSS slides the strip by exactly one copy and
// starts again, which looks like an endless row.
function fillRow(row, covers) {
  const track = row.querySelector(".cover-track");

  // One copy must be at least as wide as the screen, or a gap would show
  // at the end of the loop. 15 covers are plenty on most screens; very
  // wide ones repeat the list.
  let oneCopy = covers;
  const coverWidth = track.offsetHeight * 0.7 || 200; // covers are 7:10
  while (oneCopy.length * coverWidth < window.innerWidth) {
    oneCopy = [...oneCopy, ...covers];
  }

  const items = [...oneCopy, ...oneCopy].map(createCover);
  track.replaceChildren(...items);

  // The same speed for both rows, whatever their length: the time to
  // slide one copy's width.
  const copyWidth = track.scrollWidth / 2;
  track.style.setProperty("--row-duration", `${copyWidth / GLIDE_PX_PER_SECOND}s`);

  row.classList.add("is-ready");
}

// createCover builds one cover. While the image downloads, it shows the
// cover's own colour (dimmed by the CSS); then the image fades in.
function createCover(cover) {
  const item = document.createElement("div");
  item.className = "row-cover";
  if (cover.coverColor) {
    item.style.setProperty("--card-color", cover.coverColor);
  }

  const image = document.createElement("img");
  image.alt = ""; // decorative: nothing for screen readers to announce
  image.decoding = "async";
  // Listen for "load" before setting src, so we can't miss the event
  // when the image is already in the browser's cache.
  image.addEventListener("load", () => image.classList.add("is-loaded"), { once: true });
  image.src = cover.coverUrl;

  item.append(image);
  return item;
}

// pauseWhileHidden stops the gliding while the browser tab isn't visible,
// so it doesn't use the visitor's battery for nothing.
function pauseWhileHidden() {
  document.addEventListener("visibilitychange", () => {
    home.classList.toggle("is-paused", document.hidden);
  });
}
