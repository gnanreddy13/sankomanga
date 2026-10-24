// The home page: fills in the name and tagline, and fills the window on
// the left with popular covers (from /api/showcase) drifting up and down
// in three columns, endlessly.

import { fillBranding, preventEmptySearch } from "./common.js";

// The fewest covers a column needs to be tall enough to drift smoothly.
const MIN_COVERS_PER_COLUMN = 6;

// How often the glow and the search underline move on to the next
// cover's colour.
const COLOR_CYCLE_MS = 5000;

const showcase = document.querySelector(".showcase");
const glow = document.querySelector(".showcase-glow");

fillBranding();
preventEmptySearch(document.querySelector(".search"));
loadShowcase();
pauseWhileHidden();

// loadShowcase fetches the covers and builds the columns. The search bar
// works the whole time; the covers fade in once they're ready. If anything
// goes wrong, the window shows a soft gradient instead.
async function loadShowcase() {
  try {
    const response = await fetch("/api/showcase");
    if (!response.ok) {
      throw new Error(`showcase request failed with status ${response.status}`);
    }
    const { covers } = await response.json();
    if (covers.length === 0) {
      throw new Error("showcase has no covers");
    }

    buildColumns(covers);
    showcase.classList.add("is-ready");
    cycleColors(covers);
  } catch (error) {
    // Only decoration is missing, so log it quietly and show the fallback.
    console.warn("Showcase unavailable:", error);
    showcase.classList.add("is-fallback");
  }
}

// --- Building the columns -----------------------------------------------

// buildColumns shares the covers out between the columns and fills each
// column's track.
//
// How the endless loop works: each track holds its covers TWICE, one copy
// after the other. The CSS animation slides the track by exactly half its
// height (one full copy), at which point it looks identical to where it
// started, so the animation can jump back to the start without a visible
// seam.
function buildColumns(covers) {
  const tracks = showcase.querySelectorAll(".showcase-track");

  tracks.forEach((track, columnIndex) => {
    const columnCovers = coversForColumn(covers, columnIndex, tracks.length);
    for (const copy of [columnCovers, columnCovers]) {
      for (const cover of copy) {
        track.append(createTile(cover));
      }
    }
  });
}

// coversForColumn deals the covers out like cards (1st to column 1, 2nd to
// column 2, ...), then repeats them if there are too few to fill a column.
function coversForColumn(covers, columnIndex, columnCount) {
  let own = covers.filter((_, i) => i % columnCount === columnIndex);
  if (own.length === 0) {
    own = covers;
  }

  let result = own;
  while (result.length < MIN_COVERS_PER_COLUMN) {
    result = result.concat(own);
  }
  return result;
}

// createTile builds one cover. While the image downloads, the tile shows
// the cover's own colour (dimmed by the CSS); the image fades in on top.
function createTile(cover) {
  const tile = document.createElement("div");
  tile.className = "showcase-tile";
  if (cover.coverColor) {
    tile.style.setProperty("--tile-color", cover.coverColor);
  }

  const image = document.createElement("img");
  image.alt = ""; // decorative: nothing for screen readers to announce
  image.decoding = "async";
  // Listen for "load" before setting src, so we can't miss the event
  // when the image is already in the browser's cache.
  image.addEventListener("load", () => image.classList.add("is-loaded"), { once: true });
  image.src = cover.coverUrl;

  tile.append(image);
  return tile;
}

// --- Colours ------------------------------------------------------------

// cycleColors tints the glow behind the covers and the search underline
// with one cover's colour, then slowly moves on to the next every
// COLOR_CYCLE_MS. (The CSS makes each change a slow fade.) People who
// prefer reduced motion get the first colour only.
function cycleColors(covers) {
  const colors = covers.map((cover) => cover.coverColor).filter(Boolean);
  if (colors.length === 0) {
    return;
  }

  let index = 0;
  setLiveColor(colors[index]);
  glow.classList.add("is-on");

  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
    return;
  }
  setInterval(() => {
    index = (index + 1) % colors.length;
    setLiveColor(colors[index]);
  }, COLOR_CYCLE_MS);
}

// setLiveColor sets the --live-color CSS variable, which the glow and the
// search underline read.
function setLiveColor(color) {
  document.documentElement.style.setProperty("--live-color", color);
}

// --- Small helpers ------------------------------------------------------

// pauseWhileHidden stops the drifting while the browser tab isn't
// visible, so it doesn't use the visitor's battery for nothing.
function pauseWhileHidden() {
  document.addEventListener("visibilitychange", () => {
    showcase.classList.toggle("is-paused", document.hidden);
  });
}
