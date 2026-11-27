# Codex assignment — GopherTrunk as unified Raspberry Pi radio platform

**Priority: Architecture and migration assessment FIRST.** This document supersedes the earlier instruction to immediately implement ADS-B CPR. Do not begin a broad rewrite or migrate services before producing a source-backed decision report.

## Mission
Evaluate whether `froekjaer/GopherTrunk` should become the primary unified SDR/radio platform for ADS-B, AIS, DSC and future APRS/ACARS/other receivers on a Raspberry Pi 5 (Ubuntu 24.04, Docker), and design the smallest safe staged migration. Evaluate the option of GopherTrunk as a common event/API/UI layer while retaining specialist RF decoders.

## Current environment (reported from previous Pi inspection; reverify before relying on it)
- RTL-SDR ADS-B serial `00000010`: ultrafeeder/readsb at port 8080, Beast TCP port 30005.
- FR24 feeder radar `T-EKBI49`, local status port 8754.
- AIS-catcher SDR serial `77771111153705700`, web port 8100.
- GopherTrunk DSC SDR serial `25327425`, DSC 156.525 MHz, web port 8101; currently also consumes Beast from ultrafeeder.
- Separate radio-dashboard nginx on port 80.
- GopherTrunk working tree on Pi `~/gophertrunk-eval` contains uncommitted `internal/radio/adsb/cpr.go` and untracked `Dockerfile.pi5-web`, `internal/radio/adsb/cpr.go.bak`, `internal/radio/adsb/cpr_invalid_test.go`. They are not automatically included in this fork. Preserve all.
- Observed GopherTrunk current-aircraft API impossible surface latitude 222.798... and aggregated `crc_valid:false`; investigate separately.
- Focused ADS-B Go tests reportedly passed in Docker; this is not evidence of field parity.

## Research tasks
1. Inspect actual code, tests, docs, configs and Docker assets (not just README) for ADS-B native PPM, Beast, AIS both VHF channels, DSC, APRS, ACARS, storage, event bus, maps, APIs, alerts, health, metrics, USB handling, authentication and resource demands.
2. Explicitly distinguish **implemented**, **unit-tested**, **validated on recorded real RF**, **field-tested on Pi**, **not implemented**, and **unknown**. Do not infer maturity from documentation claims alone.
3. Assess what is needed to preserve Flightradar24 feeding, its current readsb/Beast data path, existing dashboard, raw data availability, and existing per-service UIs. Do not assume GopherTrunk itself is an FR24 feeder.
4. Evaluate AIS-catcher versus native GopherTrunk AIS GMSK reception, dual-channel reception and multi-slot message handling; identify any NMEA/AIVDM bridge or missing interoperability. Evaluate readsb versus native GopherTrunk ADS-B for sensitivity, CRC correction, overlapped preambles, aircraft completeness and CPR accuracy.
5. Evaluate resource constraints and SDR ownership: three currently assigned dongles, USB bus bandwidth, CPU/RAM, antenna/filter/LNA, per-device exclusivity, container permissions, restart behavior and serial pinning. Determine if one SDR can cover both AIS channels with GopherTrunk's actual channelization; do not assume.
6. Review safety: DSC distress alerts must not be represented as guaranteed safety-of-life monitoring; default API auth and CORS, LAN exposure, TLS, token handling, persistent volumes, backup/restore, observability and rollback.
7. Present decision matrix for (A) separate specialist services, (B) GopherTrunk as common platform with selected specialist receivers, (C) all-native GopherTrunk. Recommend a target and explicit evidence gates for each service replacement.
8. Create a prioritized, dependency-aware backlog with acceptance criteria and measurements. Include the ADS-B CPR correction from the previous task as a necessary workstream, not an assumed complete fix.

## Deliverable — phase 1
Create `docs/UNIFIED-RADIO-ARCHITECTURE-ASSESSMENT.md` with:
- Executive recommendation and decision matrix, with verified evidence and links to exact repository paths/tests
- Current vs proposed architecture and per-SDR data flow
- Capability/maturity matrix for ADS-B, FR24, AIS, DSC, APRS/ACARS and dashboard
- Explicit unknowns and non-destructive read-only Pi checks needed to close them
- Quantitative acceptance plan: frame/message counts and unique entities over matched time windows; false/invalid positions, message types, signal quality, CPU/RAM, uptime, data freshness, alert latency; compare identical RF recordings or equivalent simultaneous inputs where possible and flag non-comparable observations
- Staged migration, rollback, configuration/security, and source ownership plan
- Top three smallest safe next implementation steps and revised Codex task breakdown

## Constraints
- Work exclusively in `froekjaer/GopherTrunk`, branch `fix/adsb-cpr-position-decoding`. Do not touch `main`.
- Phase 1 is READ-ONLY analysis of existing source plus the assessment document. Do not change decoders, configuration or Docker assets yet.
- Do not access/modify Raspberry Pi, restart containers, deploy, merge, or assume access to live RF captures.
- Preserve this document and existing work. No invented benchmarks, test results or deployment claims.
- Run static/read-only tests if practical; report exactly what ran.
- Commit assessment on the branch and open a **draft PR** targeting main. Mark it architecture-only; request a separate authorization before implementing the migration or CPR changes.

## Subsequent phases (NOT authorized yet)
After review: implement ADS-B CPR correctness and regression tests; validate readsb parity; benchmark AIS and DSC; add interoperability/ingestion bridges if justified; migrate one service at a time only after explicit operator approval.

## Context
The old CPR-only task is retained in Git history, but the strategic decision now precedes implementation. The intended target is **option B** (unified GopherTrunk UI/data layer with specialist decoders where needed), subject to evidence and counterarguments.
