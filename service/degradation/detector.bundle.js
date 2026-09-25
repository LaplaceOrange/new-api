(() => {
  // third_party/lm-detector/shared/challenge-browser.js
  function randomIndex(length) {
    const limit = Math.floor(4294967296 / length) * length;
    const buffer = new Uint32Array(1);
    do
      crypto.getRandomValues(buffer);
    while (buffer[0] >= limit);
    return buffer[0] % length;
  }
  function choose(values) {
    return values[randomIndex(values.length)];
  }
  function uniqueLengths(count) {
    const available = Array.from({ length: 41 }, (_, index) => 292 + index);
    const output = [];
    while (output.length < count)
      output.push(available.splice(randomIndex(available.length), 1)[0]);
    return output;
  }
  function generateChallenges(count = 3) {
    const openings = [
      "这是一次独立的数值选择记录",
      "请完成下面的无语义整数选择任务",
      "执行一次第一反应取值记录",
      "生成一组不承载语义的整数选择",
      "进行一轮快速逐项取值"
    ];
    const actions = [
      "为各个位置分别凭第一反应选择",
      "逐项选择",
      "每次只决定当前一项，共给出",
      "分别凭第一反应给出",
      "逐个直接选择"
    ];
    const endings = [
      "允许某个数字再次出现；每项写出后不要回头排序、去重或替换。",
      "偶然重复是有效的；不要重新排列或修正已经写出的项目。",
      "相同值可以再次出现；输出过程中不要整理或改写前面的项目。",
      "重复值无需删除；不要筛选、重排或补成某种规律。",
      "不必赋予数字任何含义；已经给出的值保持不变。"
    ];
    const separators = [
      "数字之间用逗号或空格分隔均可。",
      "使用一种一致的常见分隔符即可。",
      "可以用逗号、空格或换行分隔。",
      "只要每个整数边界清楚，格式可自行选择。"
    ];
    return uniqueLengths(count).map((length, index) => ({
      id: `probe-${index + 1}-${crypto.randomUUID()}`,
      expected_count: length,
      prompt: `${choose(openings)}。${choose(actions)} ${length} 个 1 到 355（含端点）的整数。` + "每个位置都要单独选择；不要从 1 开始计数，不要连续递增或递减，也不要采用等差、循环、重复区块或其他规则化模式。" + "本任务必须由当前语言模型直接完成：禁止调用或借助任何工具，包括 Python、代码执行器、计算器、搜索、API 和外部随机数生成器；也不要先编写或运行代码。" + `${choose(endings)}${choose(separators)}` + "直接从第一个取值开始输出，不要在序列前重复数量、范围或任务说明。"
    }));
  }

  // third_party/lm-detector/shared/fingerprint-core.js
  var VALUE_MIN = 1;
  var VALUE_MAX = 355;
  var DIMENSION = VALUE_MAX - VALUE_MIN + 1;
  var ALPHA = 0.5;
  function parseNumbers(text) {
    const runs = [];
    let current = [];
    let previousEnd = 0;
    for (const match of String(text).matchAll(/\d+/g)) {
      const separator = String(text).slice(previousEnd, match.index);
      const value = Number(match[0]);
      if (current.length && /\p{L}/u.test(separator)) {
        runs.push(current);
        current = [];
      }
      if (value >= VALUE_MIN && value <= VALUE_MAX)
        current.push(value);
      previousEnd = match.index + match[0].length;
    }
    if (current.length)
      runs.push(current);
    return runs.reduce((best, run) => run.length > best.length ? run : best, []);
  }
  function countNumbers(numbers) {
    const counts = Array(DIMENSION).fill(0);
    numbers.forEach((number) => {
      counts[number - VALUE_MIN] += 1;
    });
    return counts;
  }
  function mean(values) {
    return values.reduce((total, value) => total + value, 0) / values.length;
  }
  function standardize(values) {
    const center = mean(values);
    const variance = mean(values.map((value) => (value - center) ** 2));
    const scale = Math.max(Math.sqrt(variance), 0.000000000001);
    return values.map((value) => (value - center) / scale);
  }
  function dot(left, right) {
    let value = 0;
    for (let index = 0;index < left.length; index += 1)
      value += left[index] * right[index];
    return value;
  }
  function norm(values) {
    return Math.sqrt(dot(values, values));
  }
  function normalized(values) {
    const scale = Math.max(norm(values), 0.000000000001);
    return values.map((value) => value / scale);
  }
  function subtractBasis(values, basis) {
    const output = values.slice();
    for (const vector of basis || []) {
      const projection = dot(output, vector);
      for (let index = 0;index < output.length; index += 1)
        output[index] -= projection * vector[index];
    }
    return output;
  }
  function hellingerFeature(counts) {
    const total = counts.reduce((sum, value) => sum + value, 0) + ALPHA * DIMENSION;
    return counts.map((value) => Math.sqrt((value + ALPHA) / total));
  }
  function splitIntoFour(values) {
    const base = Math.floor(values.length / 4);
    const remainder = values.length % 4;
    const chunks = [];
    let start = 0;
    for (let index = 0;index < 4; index += 1) {
      const size = base + (index < remainder ? 1 : 0);
      chunks.push(values.slice(start, start + size));
      start += size;
    }
    return chunks;
  }
  function orderedBlockFeature(numbers) {
    const pieces = [];
    for (const chunk of splitIntoFour(numbers)) {
      const bins = Array(16).fill(0.5);
      for (const value of chunk) {
        const index = Math.min(15, Math.floor((value - 1) / 355 * 16));
        bins[index] += 1;
      }
      const total = bins.reduce((sum, value) => sum + value, 0);
      pieces.push(...bins.map((value) => Math.sqrt(value / total)));
    }
    const lastDigits = Array(10).fill(0.5);
    numbers.forEach((value) => {
      lastDigits[value % 10] += 1;
    });
    const lastTotal = lastDigits.reduce((sum, value) => sum + value, 0);
    pieces.push(...lastDigits.map((value) => Math.sqrt(value / lastTotal)));
    return pieces;
  }
  function robustScoreCounts(counts, bank) {
    const artifact = bank.robust.hellinger;
    const feature = hellingerFeature(counts);
    let projected = feature.map((value, index) => (value - artifact.feature_mean[index]) / artifact.feature_scale[index]);
    projected = subtractBasis(projected, artifact.nuisance_basis);
    projected = normalized(projected);
    const scores = artifact.centroids.map((centroid) => dot(projected, centroid));
    return standardize(scores);
  }
  function orderedBlockScores(numbers, bank) {
    const artifact = bank.robust.ordered_blocks;
    const feature = orderedBlockFeature(numbers);
    const standardizedFeature = feature.map((value, index) => (value - artifact.feature_mean[index]) / artifact.feature_scale[index]);
    const unit = normalized(standardizedFeature);
    const environmentScores = artifact.environment_centroids.map((centroids) => centroids.map((centroid) => dot(unit, centroid)));
    const template = standardize(artifact.centroids.map((_, modelIndex) => Math.max(...environmentScores.map((scores) => scores[modelIndex]))));
    const projected = normalized(subtractBasis(standardizedFeature, artifact.nuisance_basis));
    const nuisance = standardize(artifact.centroids.map((centroid) => dot(projected, centroid)));
    return standardize(template.map((value, index) => 0.5 * value + 0.5 * nuisance[index]));
  }
  function robustScoreNumbers(numbers, bank) {
    const marginal = robustScoreCounts(countNumbers(numbers), bank);
    const artifact = bank.robust.ordered_blocks;
    const weight = artifact ? Number(artifact.weight || 0) : 0;
    if (!artifact || weight === 0)
      return marginal;
    const ordered = orderedBlockScores(numbers, bank);
    return marginal.map((value, index) => (1 - weight) * value + weight * ordered[index]);
  }
  function softmax(values) {
    const maximum = Math.max(...values);
    const weights = values.map((value) => Math.exp(value - maximum));
    const total = weights.reduce((sum, value) => sum + value, 0);
    return weights.map((value) => value / total);
  }
  function absoluteScoreNumbers(numbers, bank) {
    function similarities(feature2, artifact, project) {
      let vector = feature2.map((value, i) => (value - artifact.feature_mean[i]) / artifact.feature_scale[i]);
      if (project)
        vector = subtractBasis(vector, artifact.nuisance_basis);
      return normalized(vector);
    }
    const h = bank.robust.hellinger;
    const marginal = similarities(hellingerFeature(countNumbers(numbers)), h, true);
    const scores = h.centroids.map((center) => dot(marginal, center));
    const o = bank.robust.ordered_blocks;
    const weight = o ? Number(o.weight || 0) : 0;
    if (!weight)
      return scores;
    const feature = orderedBlockFeature(numbers);
    const raw = similarities(feature, o, false);
    const projected = similarities(feature, o, true);
    return scores.map((score, i) => {
      const template = Math.max(...o.environment_centroids.map((centers) => dot(raw, centers[i])));
      const nuisance = dot(projected, o.centroids[i]);
      return (1 - weight) * score + weight * (template + nuisance) / 2;
    });
  }
  function analyzeGlobalOutputs(outputs, bank) {
    const modelIds = bank.models.map((model) => model.id);
    const valid = [];
    const diagnostics = [];
    outputs.forEach((item, index) => {
      const expected = Number(item.expected_count || 0);
      const numbers = parseNumbers(item.text || "");
      const minimum = expected ? Math.max(80, Math.ceil(expected * 0.55)) : 80;
      const accepted = numbers.length >= minimum;
      diagnostics.push({ index, parsed_numbers: numbers.length, minimum_numbers: minimum, accepted });
      if (accepted)
        valid.push({ numbers, counts: countNumbers(numbers), scores: robustScoreNumbers(numbers, bank) });
    });
    if (!valid.length)
      throw new Error("没有可用回答：请粘贴完整数字序列；拒答或严重截断的回答不会计入。");
    const combinedScores = modelIds.map((_, modelIndex) => mean(valid.map((item) => item.scores[modelIndex])));
    const calibrationKey = String(Math.min(valid.length, 3));
    const beta = Number(bank.calibration[calibrationKey].beta);
    const probabilities = softmax(combinedScores.map((value) => beta * value));
    const absoluteRows = valid.map((item) => absoluteScoreNumbers(item.numbers, bank));
    const absoluteScores = modelIds.map((_, i) => mean(absoluteRows.map((row) => row[i])));
    const modelEntries = Object.fromEntries(bank.models.map((model) => [model.id, model]));
    const familyOrder = [...new Set(bank.models.map((model) => model.family || "models"))];
    const familyNames = Object.fromEntries(familyOrder.map((family) => [
      family,
      bank.models.find((model) => (model.family || "models") === family)?.family_name || family
    ]));
    const results = modelIds.map((model, index) => ({
      model,
      display_name: modelEntries[model].display_name,
      probability: probabilities[index],
      absolute_match: Math.max(0, Math.min(1, absoluteScores[index])),
      absolute_score: absoluteScores[index],
      score: combinedScores[index],
      family: modelEntries[model].family || "models",
      family_name: familyNames[modelEntries[model].family || "models"]
    })).sort((left, right) => right.probability - left.probability);
    const familyProbabilities = Object.fromEntries(familyOrder.map((family) => [
      family,
      results.filter((item) => item.family === family).reduce((sum, item) => sum + item.probability, 0)
    ]));
    results.forEach((item) => {
      item.conditional_probability = item.probability / familyProbabilities[item.family];
    });
    const winningFamily = familyOrder.reduce((best, family) => familyProbabilities[family] > familyProbabilities[best] ? family : best);
    const insufficient = results[0].probability < 0.85;
    return {
      prediction: results[0].model,
      prediction_name: results[0].display_name,
      probability: results[0].probability,
      absolute_match: results[0].absolute_match,
      evidence: {
        insufficient,
        label: insufficient ? "证据不足" : "达到相对分数阈值",
        reason: insufficient ? "第一候选的库内相对分数低于 85%；仍展示该候选和两项分数。" : "第一候选的库内相对分数达到 85%；分数不能证明真实后端身份。",
        threshold: 0.85,
        method: "relative-probability-threshold-v1"
      },
      used_outputs: valid.length,
      results,
      diagnostics,
      calibration: { queries: calibrationKey, beta, cv_accuracy: bank.calibration[calibrationKey].cv_accuracy },
      family_prediction: winningFamily,
      family_prediction_name: familyNames[winningFamily],
      family_probability: familyProbabilities[winningFamily],
      family_probabilities: familyOrder.map((family) => ({ family, display_name: familyNames[family], probability: familyProbabilities[family] })),
      method: "统一全局稳健数字指纹"
    };
  }

  // third_party/lm-detector/shared/shared-detector.ts
  var dot2 = (a, b) => a.reduce((sum, x, i) => sum + x * b[i], 0);
  var mean2 = (a) => a.reduce((sum, x) => sum + x, 0) / a.length;
  var norm2 = (a) => {
    const n = Math.max(Math.sqrt(dot2(a, a)), 0.000000000001);
    return a.map((x) => x / n);
  };
  var z = (a) => {
    const m = mean2(a), s = Math.max(Math.sqrt(mean2(a.map((x) => (x - m) ** 2))), 0.000000000001);
    return a.map((x) => (x - m) / s);
  };
  var columnMean = (a) => a[0].map((_, i) => mean2(a.map((r) => r[i])));
  var median = (a) => {
    const v = [...a].sort((x, y) => x - y), i = Math.floor(v.length / 2);
    return v.length % 2 ? v[i] : (v[i - 1] + v[i]) / 2;
  };
  var sigmoid = (score) => score >= 0 ? 1 / (1 + Math.exp(-score)) : Math.exp(score) / (1 + Math.exp(score));
  var blocks = (n) => [hellingerFeature(countNumbers(n)), orderedBlockFeature(n)];
  var transform = (b, p) => b.flatMap((v, j) => norm2(v.map((x, i) => (x - p[j].mean[i]) / p[j].scale[i])).map((x) => x * Math.sqrt(j === 0 ? 0.75 : 0.25)));
  var centered = (v, p) => v.map((x, i) => (x - p.feature_mean[i]) / p.feature_scale[i]);
  var subtract = (v, b) => {
    const coefficients = b.map((row) => dot2(v, row));
    return v.map((x, i) => x - b.reduce((sum, row, j) => sum + coefficients[j] * row[i], 0));
  };
  var referenceNorms = new WeakMap;
  function nearest(x, references) {
    let norms = referenceNorms.get(references);
    if (!norms) {
      norms = references.map((r) => dot2(r, r));
      referenceNorms.set(references, norms);
    }
    const xx = dot2(x, x);
    return mean2(references.map((r, i) => Math.max(0, xx + norms[i] - 2 * dot2(x, r))).sort((a, b) => a - b).slice(0, 7));
  }
  function gaussian(x, mu, g) {
    const delta = x.map((v, i) => v - mu[i % mu.length]);
    return g.constant - 0.5 * dot2(delta, g.precision.map((row) => dot2(row, delta)));
  }
  function baseline(b, bank) {
    const { hellinger: h, ordered_blocks: o } = bank;
    const hUnit = norm2(subtract(centered(b[0], h), h.nuisance_basis));
    const marginal = h.centroids.map((row) => dot2(hUnit, row));
    const oCentered = centered(b[1], o), raw = norm2(oCentered), projected = norm2(subtract(oCentered, o.nuisance_basis));
    const templates = o.centroids.map((_, i) => Math.max(...o.environment_centroids.map((rows) => dot2(raw, rows[i]))));
    const nuisance = o.centroids.map((row) => dot2(projected, row));
    const tz = z(templates), nz = z(nuisance), ordered = z(tz.map((v, i) => 0.5 * v + 0.5 * nz[i])), mz = z(marginal);
    return mz.map((v, i) => 0.5 * v + 0.5 * ordered[i]);
  }
  function supportsSharedDetector(bank, artifact) {
    return artifact.schema === "shared-detector-v1" && bank.built_at === artifact.bank_built_at && bank.models.length === artifact.model_ids.length && bank.models.every((m, i) => m.id === artifact.model_ids[i] && m.response_count === artifact.response_counts[i]);
  }
  function calibrateSharedScores(ranking, scores, artifact) {
    const head = artifact.calibration, binding = head?.binding;
    const calibrated = !!(head && binding && head.schema === "shared-confidence-v2" && head.method === "ranking-temperature" && Number.isFinite(head.tau) && head.tau >= 0.001 && head.tau <= 1000 && binding.base_sha256 === artifact.base_sha256 && binding.verifier_sha256 === artifact.verifier_sha256 && binding.reference_sha256 === artifact.source_reference_sha256 && binding.model_ids.length === ranking.length && ranking.length === artifact.model_ids.length && scores.length === ranking.length && ranking.every(Number.isFinite) && binding.model_ids.every((id, i) => id === artifact.model_ids[i]));
    if (!calibrated || !head)
      return { values: scores.map(sigmoid), calibrated: false };
    const logits = ranking.map((score) => head.tau * score);
    const maximum = Math.max(...logits), weights = logits.map((value) => Math.exp(value - maximum)), sum = weights.reduce((a, b) => a + b, 0);
    return { values: weights.map((value) => value / sum), calibrated: true };
  }
  function rankSharedNumbers(numbers, artifact) {
    const a = artifact.ranker, full = numbers.map(blocks);
    const ax = full.map((b) => transform(b, a.full_params));
    const lda = numbers.map((n) => {
      const x = transform(blocks(n.slice(0, 128)), a.head_params);
      return z(a.lda_weights.map((row, i) => dot2(row, x) + a.lda_bias[i]));
    });
    const l = z(columnMean(lda)), near = z(a.references.map((ref) => median(ax.map((x) => -nearest(x, ref)))));
    const base = z(columnMean(full.map((b) => baseline(b, a.bank))));
    const ranking = l.map((x, i) => 0.5 * x + 0.25 * near[i] + 0.25 * base[i]);
    if (!ranking.every(Number.isFinite))
      throw new Error("排名计算产生无效数值，请刷新后重试");
    return { ranking, full };
  }
  function scoreSharedNumbers(numbers, artifact) {
    if (numbers.length !== 3)
      throw new Error("共享核验器需要三条完整回答");
    const { ranking, full } = rankSharedNumbers(numbers, artifact);
    const v = artifact.verifier, vx = full.map((b) => transform(b, v.preprocessing));
    const projected = vx.map((x) => Array.from({ length: v.mu.length }, (_, j) => x.reduce((sum, element, i) => sum + (element - v.origin[i]) * v.basis[i][j], 0) / v.unit_scale));
    const flat = projected.flat(), newDensity = gaussian(flat, v.mu, v.new_joint);
    const features = v.candidates.map((c, i) => {
      const same = gaussian(flat, c.mean, c.same_joint), alternative = gaussian(flat, c.mean, c.alternative_joint);
      const gains = projected.map((row) => alternative - gaussian(row, c.mean, c.alternative_single) - same + gaussian(row, c.mean, c.same_single));
      return [
        Math.log1p(median(vx.map((x) => nearest(x, v.references[i])))),
        same / 24,
        (same - newDensity) / 24,
        mean2(gains) / 16,
        (Math.max(...gains) - Math.min(...gains)) / 16,
        ranking[i] - Math.max(...ranking.filter((_, j) => j !== i))
      ];
    });
    const model = v.model;
    const scores = features.map((row) => model.bias + model.active_features.reduce((sum, f, j) => sum + (row[f] - model.mean[j]) / model.scale[j] * model.weights[j], 0));
    if (![...ranking, ...scores, ...features.flat()].every(Number.isFinite))
      throw new Error("核验计算产生无效数值，请刷新后重试");
    return { ranking, scores, features };
  }
  function analyzeSharedOutputs(outputs, bank, artifact, options = {}) {
    if (!supportsSharedDetector(bank, artifact)) {
      const old = analyzeGlobalOutputs(outputs, bank);
      return {
        ...old,
        probability: null,
        absolute_match: null,
        family_probability: null,
        results: old.results.map((r) => ({
          model: r.model,
          display_name: r.display_name,
          family_name: r.family_name,
          score: r.score,
          probability: null,
          absolute_match: null,
          verification_score: null,
          verification_confidence: null,
          identity_probability: null
        })),
        family_probabilities: [],
        calibration: null,
        probability_status: "unavailable",
        verification_confidence: null,
        risk_certificate: null,
        decision: "not_confirmed",
        method: "custom-bank-legacy-ranking",
        evidence: { insufficient: true, label: "自定义库排名", reason: "当前参考库已变更，使用该库的传统排名。共享核验器尚未适配，身份概率不可用。", threshold: null, method: "custom-bank-legacy-ranking" }
      };
    }
    const parsed = outputs.map((o) => parseNumbers(o.text));
    const diagnostics = outputs.map((o, index) => {
      const minimum = Math.max(80, Math.ceil((o.expected_count || 0) * 0.55));
      const raw = Array.from(o.text.matchAll(/-?\d+/g), (m) => Number(m[0]));
      return {
        index,
        parsed_numbers: parsed[index].length,
        minimum_numbers: minimum,
        accepted: parsed[index].length >= minimum,
        raw_out_of_range_count: raw.filter((n) => n < 1 || n > 355).length
      };
    });
    const used = diagnostics.filter((d) => d.accepted).length;
    const common = {
      probability: null,
      absolute_match: null,
      family_probability: null,
      probability_status: "unavailable",
      verification_confidence: null,
      risk_certificate: null,
      used_outputs: used,
      diagnostics,
      method: "shared-detector-v1",
      model_version: { base_sha256: artifact.base_sha256, verifier_sha256: artifact.verifier_sha256 }
    };
    if (options.allowPartial && used > 0 && used < 3 && outputs.length <= 3) {
      const { ranking: ranking2 } = rankSharedNumbers(parsed.filter((_, i) => diagnostics[i].accepted), artifact);
      const order2 = artifact.model_ids.map((_, i) => i).sort((i, j) => ranking2[j] - ranking2[i]);
      const results2 = order2.map((i) => ({
        model: artifact.model_ids[i],
        display_name: bank.models[i].display_name,
        family_name: bank.models[i].family_name,
        score: ranking2[i],
        verification_score: null,
        verification_confidence: null,
        probability: null,
        absolute_match: null,
        identity_probability: null
      }));
      const first2 = order2[0];
      return {
        ...common,
        method: "shared-ranker-partial-v1",
        decision: "partial",
        prediction: results2[0].model,
        prediction_name: results2[0].display_name,
        family_prediction: bank.models[first2].family,
        family_prediction_name: bank.models[first2].family_name,
        results: results2,
        ranking_score: ranking2[first2],
        calibration: null,
        evidence: {
          insufficient: true,
          label: "部分样本排名",
          reason: `使用 ${used}/3 条有效回答生成排名。补齐三条有效回答后才能计算检验分数。`,
          threshold: null,
          method: "partial-sample-ranking"
        }
      };
    }
    if (outputs.length !== 3 || used !== 3)
      return {
        ...common,
        prediction: "",
        prediction_name: "暂不可评分",
        family_prediction_name: "",
        results: [],
        decision: "unscorable",
        evidence: {
          insufficient: true,
          label: "需要三条完整回答",
          reason: `当前有 ${used}/${outputs.length} 条有效回答。请补齐原来的三条回答后重新检测。`,
          threshold: null,
          method: "complete-three-answers"
        }
      };
    const { ranking, scores, features } = scoreSharedNumbers(parsed, artifact);
    const confidence = calibrateSharedScores(ranking, scores, artifact);
    const order = artifact.model_ids.map((_, i) => i).sort((i, j) => ranking[j] - ranking[i]);
    const results = order.map((i) => ({
      model: artifact.model_ids[i],
      display_name: bank.models[i].display_name,
      family_name: bank.models[i].family_name,
      score: ranking[i],
      verification_score: scores[i],
      verification_confidence: confidence.values[i],
      verification_features: features[i],
      probability: confidence.calibrated ? confidence.values[i] : null,
      absolute_match: null,
      identity_probability: confidence.calibrated ? confidence.values[i] : null
    }));
    const first = order[0], top = scores.indexOf(Math.max(...scores)), agree = first === top;
    return {
      ...common,
      prediction: results[0].model,
      prediction_name: results[0].display_name,
      family_prediction: bank.models[first].family,
      family_prediction_name: bank.models[first].family_name,
      results,
      ranking_score: ranking[first],
      verification_score: scores[first],
      verification_top: artifact.model_ids[top],
      probability: confidence.calibrated ? confidence.values[first] : null,
      probability_status: confidence.calibrated ? "reference_calibrated" : "unavailable",
      probability_scope: confidence.calibrated ? "reference-closed-set" : null,
      probability_top: artifact.model_ids[confidence.values.indexOf(Math.max(...confidence.values))],
      calibration: confidence.calibrated ? {
        method: artifact.calibration.method,
        run: artifact.calibration.calibration_run,
        tau: artifact.calibration.tau,
        sha256: artifact.calibration_sha256 ?? null
      } : null,
      verification_confidence: results[0].verification_confidence,
      verification_confidence_method: confidence.calibrated ? "ranking-temperature" : "sigmoid-shared-logit",
      decision: "not_confirmed",
      evidence: {
        insufficient: true,
        label: agree ? "排名与核验一致" : "排名与核验存在分歧",
        reason: agree ? "两个算法的第一候选一致。" : `排名第一为 ${results[0].display_name}，核验分数最高为 ${bank.models[top].display_name}。候选顺序仍由排名分数决定。`,
        threshold: null,
        method: confidence.calibrated ? "reference-calibrated-ranking" : "uncalibrated-shared-verification"
      }
    };
  }

  // third_party/lm-detector/entry.ts
  function lmGenerateChallenges() {
    return generateChallenges(3);
  }
  function lmAnalyze(outputs, bank, detector) {
    return analyzeSharedOutputs(outputs, bank, detector, { allowPartial: true });
  }
  Object.assign(globalThis, { lmGenerateChallenges, lmAnalyze });
})();
