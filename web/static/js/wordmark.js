// The home page's big wordmark: the name in capitals, split in two. The
// start ("sanko") is solid white and takes turns with its Japanese
// spelling (サンコー), sound by sound; the end ("manga") is a white
// outline. The CSS (home.css) draws it; this file builds it and runs the
// swap.

import { APP_NAME, GREY_WORD, JAPANESE_NAME, NAME_SOUNDS } from "./config.js";

// How long each version of the name stays before swapping: the English
// one longer, since that's the name people need to remember and type.
// The first swap waits the full LATIN_SHOWN_MS, so people read the name
// before anything moves.
const LATIN_SHOWN_MS = 6000;
const JAPANESE_SHOWN_MS = 3000;

const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

// setUpWordmark turns the plain name in the home page's heading into the
// wordmark, and loads the font for its Japanese spelling.
export function setUpWordmark() {
  buildWordmark();
  loadJapaneseFont();
}

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
