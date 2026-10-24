// The home page: fills in the name and tagline, fills two rows of popular
// covers (from /api/showcase, a different half in each row) that glide
// slowly across the top and bottom of the screen in opposite directions, and sets up the search box that
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

    // Each row gets its own half of the list, so every cover on the
    // screen is different. (With a single cover, both rows share it.)
    const half = Math.ceil(covers.length / 2);
    fillRow(topRow, covers.slice(0, half));
    fillRow(bottomRow, covers.slice(half).length > 0 ? covers.slice(half) : covers);
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
  // at the end of the loop. 15 covers are plenty on most screens (so no
  // cover shows twice); only very wide screens repeat the list.
  track.replaceChildren(...covers.map(createCover));
  while (track.scrollWidth < window.innerWidth) {
    track.append(...covers.map(createCover));
  }
  const oneCopy = [...track.children];

  // The second copy: the same covers again (the browser downloads each
  // image only once).
  track.append(...oneCopy.map((item) => item.cloneNode(true)));
  for (const image of track.querySelectorAll("img:not(.is-loaded)")) {
    whenLoaded(image);
  }

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
  image.src = cover.coverUrl;

  item.append(image);
  return item;
}

// whenLoaded fades an image in once it has downloaded. It checks
// "complete" too, in case the image was already in the browser's cache
// and finished before we started listening.
function whenLoaded(image) {
  if (image.complete && image.naturalWidth > 0) {
    image.classList.add("is-loaded");
    return;
  }
  image.addEventListener("load", () => image.classList.add("is-loaded"), { once: true });
}

// pauseWhileHidden stops the gliding while the browser tab isn't visible,
// so it doesn't use the visitor's battery for nothing.
function pauseWhileHidden() {
  document.addEventListener("visibilitychange", () => {
    home.classList.toggle("is-paused", document.hidden);
  });
}
