// The home page: sets up the big wordmark (wordmark.js), places a few
// popular manga covers (from /api/showcase) floating around it, and sets
// up the search box that opens on top of the page (search-overlay.js).
//
// The volumes stay alive in three quiet ways: each one bobs slowly (CSS),
// on desktops they shift a little with the mouse, nearer ones more (depth),
// and every few seconds one cover fades into a different popular manga.
// The covers are shuffled on every visit, so no two visits look the same.

import { fillBranding } from "./common.js";
import { setUpSearchOverlay } from "./search-overlay.js";
import { setUpWordmark } from "./wordmark.js";

const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

// How many volumes the page has room for. The CSS shows all of them on
// laptops, fewer on phones (see .volume:nth-child in home.css).
const VOLUME_COUNT = 8;

// How often one cover is swapped for another, and how long the new one
// takes to fade in over the old (the same as the fade in home.css).
const SWAP_EVERY_MS = 5000;
const CROSSFADE_MS = 1200;

// Roughly how wide a volume's cover is drawn, for the browser to pick the
// right image size: 27% of the screen on phones, at most 124px elsewhere.
const COVER_SIZES = "(max-width: 720px) 27vw, 124px";

const home = document.querySelector(".home");
const volumesLayer = document.querySelector(".volumes");

fillBranding();
setUpWordmark();
setUpSearchOverlay();
loadCovers();
pauseWhileHidden();
followMouse();

// shuffled returns a copy of the list in random order (the Fisher-Yates
// shuffle: every order is equally likely).
function shuffled(list) {
  const copy = [...list];
  for (let i = copy.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [copy[i], copy[j]] = [copy[j], copy[i]];
  }
  return copy;
}

// loadCovers fetches the covers and puts one in each volume. The search
// works the whole time; the volumes fade in once they're ready. If
// anything goes wrong, the page simply has no volumes (only decoration
// is missing).
async function loadCovers() {
  try {
    const response = await fetch("/api/showcase");
    if (!response.ok) {
      throw new Error(`showcase request failed with status ${response.status}`);
    }
    const { covers } = await response.json();
    if (covers.length === 0) {
      throw new Error("showcase has no covers");
    }

    // A different mix on every visit: the first few go into the volumes,
    // the rest wait their turn to be swapped in.
    const mixed = shuffled(covers);
    const volumes = mixed.slice(0, VOLUME_COUNT).map(createVolume);
    const waiting = mixed.slice(VOLUME_COUNT);

    volumesLayer.append(...volumes.map((volume) => volume.element));
    volumesLayer.classList.add("is-ready");

    if (!reduceMotion && waiting.length > 0) {
      swapCovers(volumes, waiting);
    }
  } catch (error) {
    console.warn("Volumes unavailable:", error);
  }
}

// createVolume builds one volume (see "the volumes" in home.css for its
// layers) with a cover in it. It returns the volume, its cover element,
// and which manga it shows now.
function createVolume(cover) {
  const element = document.createElement("div");
  element.className = "volume";

  const float = document.createElement("div");
  float.className = "volume-float";

  const coverElement = document.createElement("div");
  coverElement.className = "volume-cover";
  showCover(coverElement, cover);

  float.append(coverElement);
  element.append(float);
  return { element, coverElement, cover };
}

// showCover puts a manga's cover image on a volume. The new image is laid
// on top of any old one and fades in over it (a crossfade); once it has,
// the old one is removed. The first time, there's no old image: the
// volume shows the cover's own colour (dimmed by the CSS) until the image
// has downloaded.
//
// It offers the browser both sizes and says roughly how wide a volume is
// drawn ("sizes"), so the browser downloads just one: the 230px image on
// ordinary screens, the 460px one on sharp "Retina" screens.
// loading="lazy" means volumes the CSS hides (on phones) never download.
function showCover(coverElement, cover) {
  if (cover.coverColor) {
    coverElement.style.setProperty("--card-color", cover.coverColor);
  } else {
    coverElement.style.removeProperty("--card-color");
  }

  const image = document.createElement("img");
  image.alt = ""; // decorative: nothing for screen readers to announce
  image.decoding = "async";
  image.loading = "lazy";
  // Listen for "load" before setting the sources, so we can't miss the
  // event when the image is already in the browser's cache.
  const oldImages = [...coverElement.querySelectorAll("img")];
  image.addEventListener("load", () => {
    // Wait until the browser has drawn the image invisible once: an image
    // that was already downloaded loads instantly, and would otherwise go
    // straight to fully visible without fading.
    requestAnimationFrame(() => requestAnimationFrame(() => {
      image.classList.add("is-loaded");
      setTimeout(() => oldImages.forEach((old) => old.remove()), CROSSFADE_MS);
    }));
  }, { once: true });
  image.sizes = COVER_SIZES;
  image.srcset = coverSrcset(cover);
  image.src = cover.coverUrl; // for very old browsers without srcset

  coverElement.append(image);
}

// swapCovers fades one volume's cover into a different popular manga
// every SWAP_EVERY_MS. The manga take turns: the one leaving goes to the
// back of the waiting line, so over a visit many different ones appear.
// Only volumes that are showing count (phones hide some), a volume is
// never picked twice in a row, and nothing happens while the tab is
// hidden or the search box is open.
function swapCovers(volumes, waiting) {
  let lastPicked = null;

  setInterval(async () => {
    const searching = document.body.classList.contains("search-open");
    if (document.hidden || searching) {
      return;
    }

    const showing = volumes.filter((volume) => volume.element.offsetParent !== null);
    const choices = showing.filter((volume) => volume !== lastPicked);
    if (choices.length === 0) {
      return;
    }
    const volume = choices[Math.floor(Math.random() * choices.length)];
    lastPicked = volume;

    const next = waiting.shift();
    waiting.push(volume.cover);
    volume.cover = next;

    // Download the next cover first, so it's ready to fade straight in
    // over the old one.
    await preloadCover(next);
    showCover(volume.coverElement, next);
  }, SWAP_EVERY_MS);
}

// preloadCover downloads a cover, at the same size showCover will pick,
// and finishes when it's ready (or failed: then the old cover simply
// stays a little longer).
function preloadCover(cover) {
  return new Promise((resolve) => {
    const image = new Image();
    image.onload = resolve;
    image.onerror = resolve;
    image.sizes = COVER_SIZES;
    image.srcset = coverSrcset(cover);
    image.src = cover.coverUrl;
  });
}

// coverSrcset lists a cover's two sizes for the browser to choose from.
function coverSrcset(cover) {
  return `${cover.coverUrl} 230w, ${cover.coverUrl2x} 460w`;
}

// followMouse gives the volumes a sense of depth on desktops: as the mouse
// moves, each volume shifts a few pixels the same way, nearer ones more
// (their --depth in home.css). It sets --mx and --my (from -1 to 1, the
// mouse's position from the page's centre) and the CSS does the rest.
// Not on touch screens (no mouse), nor with "reduce motion".
function followMouse() {
  const hasMouse = window.matchMedia("(hover: hover) and (pointer: fine)").matches;
  if (reduceMotion || !hasMouse) {
    return;
  }

  let frameRequested = false;
  document.addEventListener("pointermove", (event) => {
    if (frameRequested) {
      return; // at most one update per screen refresh
    }
    frameRequested = true;
    requestAnimationFrame(() => {
      frameRequested = false;
      const x = (event.clientX / window.innerWidth) * 2 - 1;
      const y = (event.clientY / window.innerHeight) * 2 - 1;
      volumesLayer.style.setProperty("--mx", x.toFixed(3));
      volumesLayer.style.setProperty("--my", y.toFixed(3));
    });
  });
}

// pauseWhileHidden stops the volumes floating while the browser tab isn't
// visible, so it doesn't use the visitor's battery for nothing.
function pauseWhileHidden() {
  document.addEventListener("visibilitychange", () => {
    home.classList.toggle("is-paused", document.hidden);
  });
}
