# Incidents

## 2026-10-14 — Simulator exits at startup
**Symptom:** container exited immediately, asking for POD_IP.
**Cause:** with enable-kvcache on, v0.10.2 requires POD_IP (used in the KV-event topic name). Kubernetes sets it automatically; plain Docker doesn't.
**Fix:** `-e POD_IP=w1` in docker run; one distinct value per worker in compose.
**Prevention:** compose sets POD_IP for every worker.

## 2026-11-02 - "glued words" in prompts (false alarm)
Symptom: pasted loadgen output showed words like "environmentsort".
Hypothesis: tracegen joins chunks without a space.
Evidence: generator joins all have spaces; word list clean;
  grep -c environmentsort /tmp/a.jsonl → 0.
Root cause: terminal drops the space at soft-wrap points when copying long lines.
Fix: none needed.