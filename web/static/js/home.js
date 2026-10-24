// The home page: fills in the name and tagline, fills two rows of popular
// covers (from /api/showcase, a different half in each row) that glide
// slowly across the top and bottom of the screen in opposite directions, and sets up the search box that
// opens on top of the page (search-overlay.js).
//
// The gliding itself is CSS (.cover-track in style.css); this file only
// builds the rows and sets their speed.
//
// To keep the page light, each cover's image is only downloaded when the
// cover glides close to the screen (most start off-screen), and the
// browser picks the image size that fits the screen (see loadImage).

import { fillBranding } from "./common.js";
import { setUpSearchOverlay } from "./search-overlay.js";

// How fast the rows glide, in pixels per second: slow enough to read as
// calm background movement rather than something to follow.
const GLIDE_PX_PER_SECOND = 22;

// A cover's image is downloaded once the cover is within this distance of
// the screen's sides: a head start of about 14 seconds at the gliding
// speed, so images are ready before they come into view.
const LOAD_AHEAD_PX = 300;

// How often to check which covers have come near the screen.
const CHECK_EVERY_MS = 1000;

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
  let oneCopy = covers;
  track.replaceChildren(...oneCopy.map(createCover));
  while (track.scrollWidth < window.innerWidth) {
    oneCopy = [...oneCopy, ...covers];
    track.append(...covers.map(createCover));
  }

  // The second copy: the same covers again (the browser downloads each
  // image only once).
  track.append(...oneCopy.map(createCover));

  // Download each image only when its cover nears the screen.
  loadImagesNearScreen(track);

  // The same speed for both rows, whatever their length: the time to
  // slide one copy's width.
  const copyWidth = track.scrollWidth / 2;
  track.style.setProperty("--row-duration", `${copyWidth / GLIDE_PX_PER_SECOND}s`);

  row.classList.add("is-ready");
}

// createCover builds one cover, without its image yet (loadImage adds it
// later). Until the image has downloaded, the cover shows its own colour
// (dimmed by the CSS); then the image fades in.
function createCover(cover) {
  const item = document.createElement("div");
  item.className = "row-cover";
  if (cover.coverColor) {
    item.style.setProperty("--card-color", cover.coverColor);
  }
  // Remember the image addresses until it's time to download.
  item.dataset.src = cover.coverUrl;
  item.dataset.src2x = cover.coverUrl2x;
  return item;
}

// loadImagesNearScreen checks once a second where the row's covers are
// and loads the images of those near the screen, until every cover has
// its image. (The browser's IntersectionObserver, made for exactly this,
// misses covers moved by a CSS animation: the browser slides them
// without telling the page's scripts. Asking for each cover's position
// always gives where it is right now.)
function loadImagesNearScreen(track) {
  const waiting = new Set(track.children);

  const check = () => {
    for (const item of waiting) {
      const box = item.getBoundingClientRect();
      if (box.right > -LOAD_AHEAD_PX && box.left < window.innerWidth + LOAD_AHEAD_PX) {
        waiting.delete(item);
        loadImage(item);
      }
    }
    if (waiting.size === 0) {
      clearInterval(timer);
    }
  };

  const timer = setInterval(check, CHECK_EVERY_MS);
  check(); // the covers already on screen load straight away
}

// loadImage adds a cover's image. It offers the browser both sizes and
// tells it how wide the cover is drawn ("sizes"); the browser then
// downloads just one: the 230px image on ordinary screens, the 460px one
// on sharp "Retina" screens, where the small one would look blurry.
function loadImage(item) {
  const image = document.createElement("img");
  image.alt = ""; // decorative: nothing for screen readers to announce
  image.decoding = "async";
  // Listen for "load" before setting the sources, so we can't miss the
  // event when the image is already in the browser's cache.
  image.addEventListener("load", () => image.classList.add("is-loaded"), { once: true });
  image.sizes = `${item.offsetWidth}px`;
  image.srcset = `${item.dataset.src} 230w, ${item.dataset.src2x} 460w`;
  image.src = item.dataset.src; // for very old browsers without srcset

  item.append(image);
}

// pauseWhileHidden stops the gliding while the browser tab isn't visible,
// so it doesn't use the visitor's battery for nothing.
function pauseWhileHidden() {
  document.addEventListener("visibilitychange", () => {
    home.classList.toggle("is-paused", document.hidden);
  });
}
