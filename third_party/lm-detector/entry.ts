import { generateChallenges } from "./shared/challenge-browser.js";
import { analyzeSharedOutputs } from "./shared/shared-detector.ts";

function lmGenerateChallenges() {
  return generateChallenges(3);
}

function lmAnalyze(outputs, bank, detector) {
  return analyzeSharedOutputs(outputs, bank, detector, { allowPartial: true });
}

Object.assign(globalThis, { lmGenerateChallenges, lmAnalyze });
