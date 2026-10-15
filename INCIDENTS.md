# Incidents

## 2026-10-14 — Simulator exits at startup
**Symptom:** container exited immediately, asking for POD_IP.
**Cause:** with enable-kvcache on, v0.10.2 requires POD_IP (used in the KV-event topic name). Kubernetes sets it automatically; plain Docker doesn't.
**Fix:** `-e POD_IP=w1` in docker run; one distinct value per worker in compose.
**Prevention:** compose sets POD_IP for every worker.