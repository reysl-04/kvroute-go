# NOTES
## Research question
How much does history-based prefix routing improve p95 time-to-first-token (TTFT) over least-loaded routing, and at what cache pressure does the advantage disappear?

## Hypotheses (written Oct 13, before any data)
**Hypothesis 1**
> Under high prefix reuse and low cache pressure, approx_prefix beats least_loaded on p95 TTFT.

**Hypothesis 2** 
> As cache pressure rises, the advantage shrinks, because the tree predicts prefixes the worker has already evicted. This shows up as a rising false-hit rate.

**Hypothesis 3**
> Pure affinity (imbalance fallback off) raises p99 TTFT under skewed load, and the fallback recovers it.

## 2026-10-14 — Fidelity gate
**Goal:** confirm cache hits lower TTFT in the simulator.

**Setup:** llm-d-inference-sim v0.10.2, profile 8b-h100-balanced-per-token (changed: model, max-model-len, kv-cache-size)

**Result:**

| Run | Tokens | Cached | Measured |
|---|---|---|---|
| P1 cold | 3000 | 0 | 822 ms |
| P1 repeat | 3000 | 2992 | 54 ms |
| P2 control | 3000 | 0 | 802 ms |
| P3 cold | 2000 | 0 | 544 ms |
| P3 repeat | 2000 | 2000 | 49 ms |
| P4 control | 2000 | 0 | 533 ms |

PASS.

**Observations:**  
- Cache hit cut TTFT ~93%. Controls stayed slow.
- Extra time beyond the formula: 3–42 ms, largest on the first request.
- `cached_tokens` is in non-streaming responses.
- KV cache size flag: `kv-cache-size` (blocks).
- Only full 16-token blocks are cached (3000 → 2992)

**Decision:** keep the simulator; no need for the Dynamo mocker.

**Open questions:** warm-up effect on first request? `cached_tokens` in streaming responses?