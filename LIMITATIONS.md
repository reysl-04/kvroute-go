# Limitations

1. Simulator TTFT is linear in uncached tokens by construction; effect sizes depend on the chosen latency profile.
2. Simulator load model is a single linear slowdown factor up to max-num-seqs.
3. Synthetic traces only; no claim about real agentic traffic.
4. Character-level matching vs. 16-token blocks; residual error after calibration.
5. Pure-attention model assumption; hybrid Mamba/GDN models with large blocks are out of scope.
6. LRU mirror; priority-based eviction and cache pinning are out of scope.