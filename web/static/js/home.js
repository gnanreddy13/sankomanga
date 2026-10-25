// The home page: builds the big wordmark (SANKO in white, taking turns
// with サンコー, then MANGA as black-and-white manga art), fills two rows of popular covers (from
// /api/showcase) that glide slowly across the top and bottom of the screen
// in opposite directions, and sets up the search box that opens on top of
// the page (search-overlay.js).
//
// The covers are shuffled on every visit, and each row gets its own half,
// so every cover on the page is different.
//
// The gliding itself is CSS (.cover-track in style.css); this file only
// builds the rows and sets their speed.
//
// To keep the page light, each cover's image is only downloaded when the
// cover glides close to the screen (most start off-screen), and the
// browser picks the image size that fits the screen (see loadImage).

import { fillBranding } from "./common.js";
import { APP_NAME, ART_WORD, JAPANESE_NAME, NAME_SOUNDS } from "./config.js";
import { setUpSearchOverlay } from "./search-overlay.js";

// The wordmark art: an image of ART_WORD with manga art inside its
// letters, on a transparent background. If it doesn't exist, the word is
// shown as styled text instead (see .wordmark-art in style.css).
//
// Its size in pixels, so the page can keep exactly the right space for it
// before it has downloaded (only the shape, width ÷ height, matters, not
// the exact numbers). Update these whenever the image file changes.
const WORDMARK_ART = "/static/img/manga-art.png";
const WORDMARK_ART_WIDTH = 640;
const WORDMARK_ART_HEIGHT = 166;

// How long each version of the name stays before swapping: the English
// one longer, since that's the name people need to remember and type.
// The first swap waits the full LATIN_SHOWN_MS, so people read the name
// before anything moves.
const LATIN_SHOWN_MS = 6000;
const JAPANESE_SHOWN_MS = 3000;

const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

// How fast the rows glide, in pixels per second: slow enough to read as
// calm background movement rather than something to follow.
const GLIDE_PX_PER_SECOND = 22;

// A cover's image is downloaded once the cover is within this distance of
// the screen's sides: a head start of about 14 seconds at the gliding
// speed, so images are ready before they come into view.
const LOAD_AHEAD_PX = 300;

// How often to check which covers have come near the screen.
const CHECK_EVERY_MS = 1000;

// After the screen is rotated or the window resized, how long to wait for
// it to settle before rebuilding the rows (a window being dragged to a
// new size sends many resize events; this rebuilds once, at the end).
const RESIZE_SETTLE_MS = 250;

const home = document.querySelector(".home");
const topRow = document.querySelector(".cover-row--top");
const bottomRow = document.querySelector(".cover-row--bottom");

// Each row's covers, kept so the row can be rebuilt after a resize, and
// each row's image-loading timer, so a rebuild can stop the old one.
const rowCovers = new Map();
const rowLoaders = new Map();

fillBranding();
buildWordmark();
loadJapaneseFont();
setUpSearchOverlay();
loadCovers();
pauseWhileHidden();
refillRowsAfterResize();

// --- The wordmark ----------------------------------------------------------

// buildWordmark splits the name in the heading into two parts: the start
// ("sanko") stays plain white and takes turns with JAPANESE_NAME
// (サンコー), and ART_WORD at the end ("manga") becomes the art: the image
// once it has loaded, the styled text until then (or for good, if there's
// no image). The CSS shows the wordmark in capitals.
//
// Screen readers might read a name split into parts oddly, so the visible
// wordmark is hidden from them and the plain name is read instead.
function buildWordmark() {
  const heading = document.querySelector(".brand-name");
  const hasArtWord = ART_WORD !== "" && APP_NAME.endsWith(ART_WORD);
  const plainPart = hasArtWord ? APP_NAME.slice(0, -ART_WORD.length) : APP_NAME;

  const forScreenReaders = document.createElement("span");
  forScreenReaders.className = "visually-hidden";
  forScreenReaders.textContent = APP_NAME;

  const visible = document.createElement("span");
  visible.className = "wordmark";
  visible.setAttribute("aria-hidden", "true");
  visible.append(createNameSwap(plainPart));

  if (hasArtWord) {
    visible.append(createArt());
  }

  heading.replaceChildren(forScreenReaders, visible);
}

// createArt builds the art image for ART_WORD. It goes into the name
// straight away, with its width and height set, so the browser keeps
// exactly the right space for it while it downloads: nothing around it
// moves when it arrives. Until then it's invisible (.is-loading); then it
// fades into its space. If the browser already has it, it's simply there.
//
// If the image is missing (say, a fresh copy of the project without the
// art), the styled word takes its place. That's known within moments,
// while the name is still rising into view.
function createArt() {
  const image = new Image(WORDMARK_ART_WIDTH, WORDMARK_ART_HEIGHT);
  image.className = "wordmark-art-image is-loading";
  image.alt = ""; // the heading's hidden text already says the name
  image.decoding = "async";

  image.addEventListener("load", () => image.classList.remove("is-loading"), { once: true });
  image.addEventListener("error", () => image.replaceWith(createStyledWord()), { once: true });
  image.src = WORDMARK_ART;

  if (image.complete && image.naturalWidth > 0) {
    // Already downloaded on an earlier visit: no need to fade.
    image.classList.remove("is-loading");
  }
  return image;
}

// createStyledWord builds ART_WORD as styled text, for when there's no
// art image (see .wordmark-art in style.css).
function createStyledWord() {
  const word = document.createElement("span");
  word.className = "wordmark-art";
  word.textContent = ART_WORD;
  return word;
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

// --- The rows of covers ----------------------------------------------------

// loadCovers fetches the covers and fills both rows. The search works the
// whole time; the rows fade in once they're ready. If anything goes wrong,
// the rows simply stay empty (only decoration is missing).
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

    // A different mix on every visit, and each row gets its own half of
    // the list, so every cover on the screen is different. (With a single
    // cover, both rows share it.)
    const mixed = shuffled(covers);
    const half = Math.ceil(mixed.length / 2);
    fillRow(topRow, mixed.slice(0, half));
    fillRow(bottomRow, mixed.length > 1 ? mixed.slice(half) : mixed);
  } catch (error) {
    console.warn("Cover rows unavailable:", error);
  }
}

// fillRow puts the covers into a row's strip, then a second identical
// copy right after them. The CSS slides the strip by exactly one copy and
// starts again, which looks like an endless row.
function fillRow(row, covers) {
  const track = row.querySelector(".cover-track");
  rowCovers.set(row, covers);
  clearInterval(rowLoaders.get(row)); // stop loading images for an older strip

  // One copy must be at least as wide as the screen, or a gap would show
  // at the end of the loop. 15 covers are plenty on most screens (so no
  // cover shows twice); only very wide screens repeat the list. The limit
  // of 10 copies is a safety net: if the covers ever had no width (say, a
  // very short screen), the strip would never get wider and this would
  // otherwise repeat forever.
  let oneCopy = covers;
  track.replaceChildren(...oneCopy.map(createCover));
  for (let copies = 1; copies < 10 && track.scrollWidth < window.innerWidth; copies++) {
    oneCopy = [...oneCopy, ...covers];
    track.append(...covers.map(createCover));
  }

  // The second copy: the same covers again (the browser downloads each
  // image only once).
  track.append(...oneCopy.map(createCover));

  // Download each image only when its cover nears the screen.
  rowLoaders.set(row, loadImagesNearScreen(track));

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
  return timer;
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

// refillRowsAfterResize rebuilds both rows once the screen has been
// rotated or the window resized. The covers' size follows the screen (the
// CSS does that), but how many copies a strip needs to cover the screen's
// width, and how long one loop takes at the gliding speed, were worked
// out for the old size: a phone turned sideways can be twice as wide with
// covers half as tall, which would leave a gap at the end of the loop.
// (The images are already downloaded, so a rebuild is quick.)
function refillRowsAfterResize() {
  let lastWidth = window.innerWidth;
  let lastHeight = window.innerHeight;
  let settleTimer = null;

  window.addEventListener("resize", () => {
    clearTimeout(settleTimer);
    settleTimer = setTimeout(() => {
      // Phones also send "resize" when their toolbars slide in or out; a
      // small change in height alone isn't worth a rebuild.
      const widthChanged = window.innerWidth !== lastWidth;
      const heightChangedALot = Math.abs(window.innerHeight - lastHeight) > 120;
      if (!widthChanged && !heightChangedALot) {
        return;
      }
      lastWidth = window.innerWidth;
      lastHeight = window.innerHeight;

      for (const [row, covers] of rowCovers) {
        fillRow(row, covers);
      }
    }, RESIZE_SETTLE_MS);
  });
}

// pauseWhileHidden stops the gliding while the browser tab isn't visible,
// so it doesn't use the visitor's battery for nothing.
function pauseWhileHidden() {
  document.addEventListener("visibilitychange", () => {
    home.classList.toggle("is-paused", document.hidden);
  });
}
