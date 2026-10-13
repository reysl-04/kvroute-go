## Research question
How much does history-based prefix routing improve p95 time-to-first-token (TTFT) over least-loaded routing, and at what cache pressure does the advantage disappear?

### Hypothesis 1
| Under high prefix reuse and low cache pressure, approx_prefix beats least_loaded on p95 TTFT.
### Hypothesis 2 
| As cache pressure rises, the advantage shrinks, because the tree predicts prefixes the worker has already evicted. This shows up as a rising false-hit rate.
### Hypothesis 3
| Pure affinity (imbalance fallback off) raises p99 TTFT under skewed load, and the fallback recovers it.