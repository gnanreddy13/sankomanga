// The app's name and tagline. This is the only place the frontend spells
// them out: change them here and every page updates.
// (The backend has its own name constant, appName, in config.go.)

export const APP_NAME = "sankomanga";
export const TAGLINE = "explore your next manga";

// The wordmark (the big name on the home page, and the small one at the
// top of the other pages; see buildWordmark in home.js and
// fillMiniWordmark in common.js):
// - GREY_WORD: the end of APP_NAME shown in grey, after the start in
//   white (SANKO in white, MANGA in grey). If APP_NAME doesn't end with
//   it, the whole name is white.
// - NAME_SOUNDS: the start of the name ("sanko") split into its sounds,
//   each with its katakana. On the home page the name takes turns with
//   its Japanese spelling, サンコー, sound by sound: SA becomes サ, N
//   becomes ン, KO becomes コー. サンコー reads "sankō", the sound of 参考
//   ("reference"). Decoration only: screen readers skip it, and other
//   pages don't show it. Set it to [] to keep the name still.
//   (If the sounds don't spell the start of the name, the whole word
//   swaps at once instead.)
export const GREY_WORD = "manga";
export const NAME_SOUNDS = [
  ["sa", "サ"],
  ["n", "ン"],
  ["ko", "コー"],
];
export const JAPANESE_NAME = NAME_SOUNDS.map(([, katakana]) => katakana).join("");
