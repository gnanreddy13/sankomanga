// The home page: fills in the name and tagline, deals a fan of popular
// covers (from /api/showcase) rising from the bottom of the screen, and
// sets up the search box that opens on top of the page
// (search-overlay.js).
//
// The fan stays alive in two calm ways: it slowly "breathes" (spreads and
// closes a few degrees; that's CSS), and every few seconds one cover slides
// down out of the fan and a different popular cover slides up in its place.

import { fillBranding } from "./common.js";
import { setUpSearchOverlay } from "./search-overlay.js";

// How many covers are in the fan: fewer on narrow phone screens.
const IS_NARROW = window.matchMedia("(max-width: 720px)").matches;
const FAN_SIZE = IS_NARROW ? 5 : 7;

// The angle between neighbouring covers, in degrees.
const FAN_STEP_DEG = IS_NARROW ? 11 : 9;

// How often one cover is swapped for another, and how long it takes to
// slide out (or back in).
const SWAP_EVERY_MS = 4000;
const SLIDE_MS = 700;

const fan = document.querySelector(".fan");
const cardsLayer = document.querySelector(".fan-cards");
const glow = document.querySelector(".fan-glow");
const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

fillBranding();
setUpSearchOverlay();
loadFan();
pauseWhileHidden();

// loadFan fetches the covers and deals the fan. The search works the whole
// time; the fan fades in once it's ready. If anything goes wrong, the lower
// half simply stays dark (only decoration is missing).
async function loadFan() {
  try {
    const response = await fetch("/api/showcase");
    if (!response.ok) {
      throw new Error(`showcase request failed with status ${response.status}`);
    }
    const { covers } = await response.json();
    if (covers.length === 0) {
      throw new Error("showcase has no covers");
    }

    // The first few covers go into the fan; the rest wait their turn to
    // be swapped in.
    const inFan = covers.slice(0, FAN_SIZE);
    const waiting = covers.slice(FAN_SIZE);

    const slots = inFan.map((cover, index) => createCard(cover, index, inFan.length));
    cardsLayer.append(...slots.map((slot) => slot.card));
    fan.classList.add("is-ready");

    // The glow starts in the colour of the middle cover.
    setGlow(inFan[Math.floor(inFan.length / 2)].coverColor);

    if (!reduceMotion && waiting.length > 0) {
      startSwapping(slots, waiting);
    }
  } catch (error) {
    console.warn("Fan unavailable:", error);
  }
}

// --- Building the fan -----------------------------------------------------

// createCard builds one card of the fan. The outer card is tilted to its
// place in the fan (and "breathes", in the CSS); the inner face is the part
// that slides out and back in when the cover is swapped.
//
// It returns the card, its face, and which cover it shows now.
function createCard(cover, index, count) {
  // How far from the middle this card is: e.g. -3 … 3 for 7 cards.
  const offset = index - (count - 1) / 2;

  const card = document.createElement("div");
  card.className = "fan-card";
  card.style.setProperty("--angle", `${offset * FAN_STEP_DEG}deg`);
  // The middle card lies on top, the outer ones underneath.
  card.style.zIndex = String(Math.round(count - Math.abs(offset)));

  const face = document.createElement("div");
  face.className = "fan-card-face";
  showCover(face, cover);

  card.append(face);
  return { card, face, cover };
}

// showCover puts a cover on a card's face. While the image downloads, the
// face shows the cover's own colour (dimmed by the CSS); then it fades in.
function showCover(face, cover) {
  if (cover.coverColor) {
    face.style.setProperty("--card-color", cover.coverColor);
  } else {
    face.style.removeProperty("--card-color");
  }

  const image = document.createElement("img");
  image.alt = ""; // decorative: nothing for screen readers to announce
  image.decoding = "async";
  // Listen for "load" before setting src, so we can't miss the event
  // when the image is already in the browser's cache.
  image.addEventListener("load", () => image.classList.add("is-loaded"), { once: true });
  image.src = cover.coverUrl;

  face.replaceChildren(image);
}

// --- Swapping covers ------------------------------------------------------

// startSwapping slides one cover out of the fan and another in, every
// SWAP_EVERY_MS. The covers take turns: the one leaving goes to the back
// of the waiting line, so over time every popular cover appears.
function startSwapping(slots, waiting) {
  let busy = false;
  let lastSlotIndex = -1;

  setInterval(async () => {
    if (busy || document.hidden) {
      return; // don't overlap swaps, or swap while nobody's looking
    }
    busy = true;

    // A random card, but never the same one twice in a row.
    let slotIndex;
    do {
      slotIndex = Math.floor(Math.random() * slots.length);
    } while (slotIndex === lastSlotIndex && slots.length > 1);
    lastSlotIndex = slotIndex;

    const slot = slots[slotIndex];
    const next = waiting.shift();

    // Download the next cover first, so it slides in fully drawn.
    await preloadImage(next.coverUrl);

    // Slide down out of the fan…
    await slot.face.animate(
      [
        { transform: "translateY(0)", opacity: 1 },
        { transform: "translateY(60%)", opacity: 0 },
      ],
      { duration: SLIDE_MS, easing: "cubic-bezier(0.55, 0, 0.75, 0.4)", fill: "forwards" },
    ).finished;

    // …swap the picture while it's out of sight…
    waiting.push(slot.cover);
    slot.cover = next;
    showCover(slot.face, next);
    setGlow(next.coverColor);

    // …and slide back up into place.
    await slot.face.animate(
      [
        { transform: "translateY(60%)", opacity: 0 },
        { transform: "translateY(0)", opacity: 1 },
      ],
      { duration: SLIDE_MS, easing: "cubic-bezier(0.22, 1, 0.36, 1)", fill: "forwards" },
    ).finished;

    busy = false;
  }, SWAP_EVERY_MS);
}

// preloadImage downloads an image and finishes when it's ready (or failed:
// a broken cover just shows its colour, which is fine).
function preloadImage(url) {
  return new Promise((resolve) => {
    const image = new Image();
    image.onload = resolve;
    image.onerror = resolve;
    image.src = url;
  });
}

// --- Colour and small helpers ---------------------------------------------

// setGlow tints the soft glow behind the fan with a cover's colour, by
// setting the --live-color CSS variable (the CSS fades between colours).
function setGlow(color) {
  if (color) {
    document.documentElement.style.setProperty("--live-color", color);
  }
  glow.classList.add("is-on");
}

// pauseWhileHidden stops the breathing while the browser tab isn't
// visible, so it doesn't use the visitor's battery for nothing. (Swaps
// skip themselves while hidden.)
function pauseWhileHidden() {
  document.addEventListener("visibilitychange", () => {
    fan.classList.toggle("is-paused", document.hidden);
  });
}
