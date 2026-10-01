// Build-time verification of the exact package shipped in both Docker images.
// These deterministic samples are test data, not a copy of the detector.
const proc = Bun.spawnSync([
  process.execPath, "node_modules/lmfpd/bin/fpd.js",
  "--input", "samples.json", "--json", "--no-update-check",
], { stdout: "pipe", stderr: "pipe", timeout: 60_000 });
if (proc.exitCode !== 0) throw new Error("Bundled lmfpd cannot score offline samples");
const result = JSON.parse(new TextDecoder().decode(proc.stdout));
if (result.schema !== "fpd-detection-v1" || result.cancelled ||
    result.requested_rounds !== 1 || result.completed_rounds !== 1 || result.scored_rounds !== 1 ||
    result.rounds?.length !== 1 || result.rounds[0].error || result.rounds[0].samples?.length !== 3 ||
    result.rounds[0].samples.some(s => !["complete", "truncated"].includes(s.state)) ||
    !result.analysis?.prediction || !result.analysis?.decision || result.analysis.decision === "unscorable" ||
    !/^[a-f0-9]{64}$/.test(result.bank?.reference_sha256) || !(result.bank?.models > 0)) {
  throw new Error("Bundled lmfpd returned an incompatible result");
}
console.log(`Verified lmfpd bank=${result.bank.reference_sha256} models=${result.bank.models}`);
