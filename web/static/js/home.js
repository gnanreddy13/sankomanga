// The home page: builds the big wordmark (SANKO in white, taking turns
// with サンコー, then MANGA in grey), places a few popular manga covers
// (from /api/showcase) floating around it, and sets up the search
// box that opens on top of the page (search-overlay.js).
//
// The volumes stay alive in three quiet ways: each one bobs slowly (CSS),
// on desktops they shift a little with the mouse, nearer ones more (depth),
// and every few seconds one cover fades into a different popular manga.
// The covers are shuffled on every visit, so no two visits look the same.

import { fillBranding } from "./common.js";
import { APP_NAME, GREY_WORD, JAPANESE_NAME, NAME_SOUNDS } from "./config.js";
import { setUpSearchOverlay } from "./search-overlay.js";

// How long each version of the name stays before swapping: the English
// one longer, since that's the name people need to remember and type.
// The first swap waits the full LATIN_SHOWN_MS, so people read the name
// before anything moves.
const LATIN_SHOWN_MS = 6000;
const JAPANESE_SHOWN_MS = 3000;

const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

// How many volumes the page has room for. The CSS shows all of them on
// laptops, fewer on phones (see .volume:nth-child in style.css).
const VOLUME_COUNT = 8;

// How often one cover is swapped for another, and how long the new one
// takes to fade in over the old (the same as the fade in style.css).
const SWAP_EVERY_MS = 5000;
const CROSSFADE_MS = 1200;

// Roughly how wide a volume's cover is drawn, for the browser to pick the
// right image size: 27% of the screen on phones, at most 124px elsewhere.
const COVER_SIZES = "(max-width: 720px) 27vw, 124px";

const home = document.querySelector(".home");
const volumesLayer = document.querySelector(".volumes");

fillBranding();
buildWordmark();
loadJapaneseFont();
setUpSearchOverlay();
loadCovers();
pauseWhileHidden();
followMouse();

// --- The wordmark ----------------------------------------------------------

// buildWordmark splits the name in the heading into two parts: the start
// ("sanko") in white, taking turns with JAPANESE_NAME (サンコー), and
// GREY_WORD at the end ("manga") in grey, like the small wordmark on the
// other pages. The CSS shows the wordmark in capitals.
//
// Screen readers might read a name split into parts oddly, so the visible
// wordmark is hidden from them and the plain name is read instead.
function buildWordmark() {
  const heading = document.querySelector(".brand-name");
  const hasGreyWord = GREY_WORD !== "" && APP_NAME.endsWith(GREY_WORD);
  const plainPart = hasGreyWord ? APP_NAME.slice(0, -GREY_WORD.length) : APP_NAME;

  const forScreenReaders = document.createElement("span");
  forScreenReaders.className = "visually-hidden";
  forScreenReaders.textContent = APP_NAME;

  const visible = document.createElement("span");
  visible.className = "wordmark";
  visible.setAttribute("aria-hidden", "true");
  visible.append(createNameSwap(plainPart));

  if (hasGreyWord) {
    const grey = document.createElement("span");
    grey.className = "wordmark-grey";
    grey.textContent = GREY_WORD;
    visible.append(grey);
  }

  heading.replaceChildren(forScreenReaders, visible);
}

// createNameSwap builds the spot where the start of the name ("sanko")
// and its Japanese spelling take turns: both are there, one on top of the
// other, and the CSS shows one at a time.
//
// Each version is split into its sounds (NAME_SOUNDS), one element per
// sound, numbered with --sound. The CSS staggers the swap by that number,
// so the sounds change one after another, left to right: SA → サ, N → ン,
// KO → コー.
function createNameSwap(latinName) {
  const swap = document.createElement("span");
  swap.className = "wordmark-swap";

  // The sounds must spell the start of the name; if they don't (say the
  // name changed but the sounds didn't), swap the whole word at once.
  const spellsName = NAME_SOUNDS.map(([latin]) => latin).join("") === latinName;
  const sounds = spellsName ? NAME_SOUNDS : [[latinName, JAPANESE_NAME]];

  const latin = document.createElement("span");
  latin.className = "wordmark-latin";
  const japanese = document.createElement("span");
  japanese.className = "wordmark-japanese";

  sounds.forEach(([latinSound, katakana], index) => {
    latin.append(createSound(latinSound, index));
    japanese.append(createSound(katakana, index));
  });

  swap.append(latin, japanese);

  if (!reduceMotion && JAPANESE_NAME !== "") {
    takeTurns(swap);
  }
  return swap;
}

// createSound builds one sound of the name, numbered for the stagger.
function createSound(text, index) {
  const sound = document.createElement("span");
  sound.textContent = text;
  sound.style.setProperty("--sound", String(index));
  return sound;
}

// takeTurns swaps the two versions of the name back and forth: the
// English one for LATIN_SHOWN_MS, then the Japanese one for
// JAPANESE_SHOWN_MS, and so on. While the tab is hidden or the search box
// is open, it just waits (the name is hidden or blurred then).
function takeTurns(swap) {
  const showingJapanese = swap.classList.contains("is-japanese");
  const wait = showingJapanese ? JAPANESE_SHOWN_MS : LATIN_SHOWN_MS;

  setTimeout(() => {
    const searching = document.body.classList.contains("search-open");
    if (!document.hidden && !searching) {
      swap.classList.toggle("is-japanese");
    }
    takeTurns(swap);
  }, wait);
}

// loadJapaneseFont loads the font for JAPANESE_NAME (Noto Sans JP, extra
// heavy) from Google Fonts. "text=" asks for just the few characters we
// use, so the download is tiny instead of a whole Japanese font. Until it
// arrives, the device's own Japanese font is used.
function loadJapaneseFont() {
  const font = document.createElement("link");
  font.rel = "stylesheet";
  font.href =
    "https://fonts.googleapis.com/css2?family=Noto+Sans+JP:wght@900" +
    `&text=${encodeURIComponent(JAPANESE_NAME)}&display=swap`;
  document.head.append(font);
}

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

// --- The volumes -----------------------------------------------------------

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

// createVolume builds one volume (see "the volumes" in style.css for its
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
// (their --depth in style.css). It sets --mx and --my (from -1 to 1, the
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
