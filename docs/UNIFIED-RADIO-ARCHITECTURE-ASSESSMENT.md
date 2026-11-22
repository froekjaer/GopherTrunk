# Unified radio architecture assessment — GopherTrunk on the Raspberry Pi 5

**Status:** Phase 1 deliverable for [`docs/ADSB-CPR-CODEX-TASK.md`](ADSB-CPR-CODEX-TASK.md). Architecture and evidence only.
**Branch:** written on top of `fix/adsb-cpr-position-decoding` at `3d76480`.
**Scope boundary:** no decoder, configuration, Docker or Pi changes were made. Nothing was deployed, restarted or merged. The Raspberry Pi was not accessed. No live RF captures were available.
**Date:** 2026-11-22

---

## 0. How to read this document

Every capability claim carries one of these evidence levels. A claim is only as strong as its label.

| Label | Meaning |
|---|---|
| **IMPL** | Code exists and was read in this repository at the stated path. |
| **UNIT** | Covered by a unit/synthetic test that ran green in this assessment (see §11). A green test says nothing about real air. |
| **REC-RF** | Validated against a recording of real RF (a committed fixture or a reference decoder's real recording). |
| **FIELD** | Observed working on the Pi in this deployment. **No GopherTrunk capability in this document has a FIELD label** — no field evidence was provided beyond the observations quoted in the task. |
| **NOT-IMPL** | Searched for and not found. |
| **DEFECT** | A reproduced or code-proven defect (how it was shown is stated). |
| **UNKNOWN** | Cannot be settled from source; needs a Pi check or a capture (§7). |
| **ASSUMPTION** | Taken from the task's environment description, not reverified. |

"Specialist" means the existing purpose-built decoders: readsb/ultrafeeder (ADS-B), the FR24 feeder, and AIS-catcher. External facts about those tools are cited and marked as such. They were not re-measured here.

---

## 1. Executive recommendation

**Adopt a constrained form of Option B.** GopherTrunk should become the common **event, storage, API and UI layer**. The specialist receivers keep **sole ownership of the ADS-B and AIS SDRs and RF decoding**. GopherTrunk ingests their output read-only.

This agrees with the task's stated target, but the evidence moves it in three ways the task did not anticipate:

1. **GopherTrunk is not yet fit to be the system of record, even for ingested data.** Two ingest-side defects corrupt the ADS-B data GopherTrunk already shows:
   - CPR decoding accepts impossible latitudes.
   - Surface positions are decoded with the airborne formula.
   - The "current aircraft" API always reports `crc_valid:false`.

   These are reproduced or code-proven (§4.1). They are the likely explanation for the `latitude 222.798…` and `crc_valid:false` observations in the task. Fixing them is a **precondition** for Option B, not a parallel nice-to-have.

2. **GopherTrunk's DSC decoder uses the wrong character code, so it cannot validate real ITU-R M.493 traffic.**
   - It checks each 10-bit character with a CRC-3 (`g(x)=x³+x+1`, [`internal/radio/dsc/bch.go`](../internal/radio/dsc/bch.go)).
   - ITU-R M.493 defines the three check bits as the **count of B (zero) elements** in the seven information bits.
   - Only **17 of 128** valid M.493 characters pass GopherTrunk's check. The phasing character DX=125 is not among them, and the receiver locks onto exactly that character (§4.3).
   - **The DSC SDR currently assigned to GopherTrunk (`25327425`) is therefore very unlikely to be producing valid real-air decodes.** This is inferred from code, not observed. Close it with Pi check P-7 before anything else.
   - DSC must not be treated as a monitoring capability of any kind until this is fixed and validated on a recording.

3. **For AIS, the cost of switching to native GopherTrunk is architectural, not just a question of proof.**
   - GopherTrunk decodes **one AIS channel per SDR**. Each `ais.channels` entry retunes its dongle.
   - There is **no AIVDM/NMEA output** at all.
   - Today AIS-catcher covers both channels on one dongle and feeds UDP partners. Going native would need a **fourth dongle** and new NMEA output code before reaching parity, before any reception-quality comparison.
   - AIS-catcher should stay the AIS receiver for the foreseeable future.

**Option C (all-native GopherTrunk) is rejected for now.** It is blocked independently by:
- FR24 feeding: GopherTrunk is not an FR24 feeder.
- Missing AIS NMEA output and dual-channel support.
- A non-conformant DSC code.
- Zero field or recorded-RF validation of the native ADS-B, AIS and DSC RF paths.

### Decision in one line
> Keep readsb + FR24 + AIS-catcher as RF owners. Make GopherTrunk a read-only consumer and aggregator once its ingest and integrity defects are fixed. Bring DSC up natively only after the character code is corrected and proven on a real Ch-70 recording. Add APRS/ACARS later on *new* dongles behind the same evidence gates.

---

## 2. Decision matrix

Scoring: ● strong / ◐ partial / ○ weak or blocked. "Evidence" cites the section where the claim is backed.

| Criterion | A — Separate specialist services | B — GT common layer + specialist receivers (recommended) | C — All-native GT |
|---|---|---|---|
| ADS-B reception quality | ● readsb: CRC single-bit fix by default, range/speed checks (external docs [readsb man page]) | ● unchanged — readsb keeps the dongle | ○ native PPM is 2 Msps magnitude-only, no bit-error correction, no phase correction (§4.1) — UNIT only |
| FR24 continuity | ● | ● unchanged (FR24 keeps its Beast path) | ○ GT has no FR24 feeder; would still need fr24feed + Beast source (§4.2) |
| AIS reception + partner feeds | ● AIS-catcher dual-channel on one dongle, NMEA over UDP (external docs) | ● unchanged; GT would ingest NMEA (bridge NOT-IMPL, §4.4) | ○ one channel per dongle, no NMEA out, GMSK front end never run on real air (§4.4) |
| DSC | ◐ currently GT-owned but code non-conformant; previous Python decoder parked (no symbol sync) | ◐ same, until DSC fix is validated (§4.3) | ◐ same |
| Single map/API/history | ○ three UIs, no shared store | ● after integrity fixes | ● after integrity fixes |
| Data integrity today | ● per tool | ○ → ● gated on §4.1 / §4.5 fixes | ○ |
| Safety framing (DSC) | ◐ | ◐ needs explicit "not GMDSS watch-keeping" posture + alerting (§5) | ◐ |
| SDR ownership clarity | ● 1 dongle = 1 owner | ● same; GT owns DSC only | ◐ GT owns all; AIS needs a 4th dongle |
| Blast radius of one fault | ● isolated | ◐ GT outage loses the unified view only, not feeds | ○ one process outage loses everything incl. FR24 data source |
| Resource use on Pi 5 | ● known-good today (ASSUMPTION) | ◐ + GT ingest/SQLite write load — UNKNOWN, measure (§6) | ◐ UNKNOWN; pure-Go DSP on one process |
| Migration risk | ● none | ◐ low, reversible per stage | ○ high, several irreversible-feeling cutovers |
| Maintenance / source ownership | ◐ several upstreams | ◐ several upstreams + a fork of a fast-moving repo (§9) | ○ fork carries all RF risk |

**Net:** B dominates on unified data and keeps A's reception and feeds. C has no criterion where it is strictly better than B today.

---

## 3. Architecture

### 3.1 Current (ASSUMPTION — from the task; reverify with §7)

```
                 ┌──────────── Raspberry Pi 5 (Ubuntu 24.04, Docker) ───────────┐
1090 MHz ─ SDR 00000010 ─▶ ultrafeeder/readsb ─┬─▶ :8080 web (tar1090)            │
                                               ├─▶ :30005 Beast ─┬─▶ fr24feed (T-EKBI49, :8754)
                                               │                 └─▶ GopherTrunk (beast_upstreams)
162 MHz ── SDR 77771111153705700 ─▶ AIS-catcher ─┬─▶ :8100 web                     │
                                                  └─▶ UDP NMEA → partners (x4?)    │
156.525 ── SDR 25327425 ─▶ GopherTrunk DSC ─▶ :8101 web/API                       │
                         radio-dashboard nginx :80                                  │
                 └──────────────────────────────────────────────────────────────────┘
```

Discrepancy to resolve: earlier project notes describe LXD/LXC containers and a DSC dongle with an FC0012 tuner. The task says Docker. P-1 settles which runtime and which dongles are actually in use.

### 3.2 Target (Option B, constrained)

```
1090 ─ SDR A ─▶ readsb/ultrafeeder ─▶ :30005 Beast ─┬─▶ fr24feed            (unchanged, RF owner)
                                    └─▶ aircraft.json ┴─▶ GT beast_upstreams (read-only, CPR fixed,
                                                                             or positions taken from readsb)
162  ─ SDR B ─▶ AIS-catcher (A+B) ─┬─▶ UDP partners    (unchanged, RF owner)
                                   └─▶ UDP/TCP NMEA ──▶ GT NMEA ingest (NEW, read-only)
156.525 ─ SDR C ─▶ GT DSC (native, only after DSC-1..3 pass) ─▶ GT store/API/UI
144.800 ─ SDR D (future, new dongle) ─▶ GT APRS (after APRS gate)
131.x   ─ SDR E (future) ─▶ acarsdec  or  GT ACARS (after ACARS gate)

GT: SQLite store · /api/v1/{adsb,ais,dsc,…} · unified web UI · alerts · /metrics
     bound to LAN with auth on reads, or behind the existing nginx :80
```

### 3.3 Per-SDR data flow and ownership

| SDR | Owner (target) | Who may retune it | GT relationship | Failure effect if GT dies |
|---|---|---|---|---|
| `00000010` 1090 MHz | readsb (ultrafeeder) | readsb only | Beast TCP client, read-only (`internal/radio/adsb/beast`) | None on FR24; GT map stale |
| `77771111153705700` 162 MHz | AIS-catcher | AIS-catcher only | NMEA client (to be built) | None on partner feeds |
| `25327425` 156.525 MHz | GopherTrunk | GT only, pinned `role: dedicated` | Native decoder | DSC view lost |
| future | per evidence gate | — | — | — |

Rule: **no SDR is shared between processes and no GT decoder entry may reference a specialist-owned serial.** GT pins single-channel dongles as `sdr.RoleDedicated` (`markDedicatedHints`, see CLAUDE.md #1316 note). That protects against GT's own voice pool retuning them. It does not stop a misconfigured `adsb.channels` / `ais.channels` entry from claiming a specialist's dongle. That is a config-review check (§8).

---

## 4. Capability and maturity findings

### 4.1 ADS-B

| Capability | Level | Evidence |
|---|---|---|
| Mode-S decode DF0/4/5/11/16/17/18/20/21, CRC-24 | IMPL, UNIT | `internal/radio/adsb/adsb.go`, `parse.go`; canonical dump1090 / mode-s.org vectors in `adsb_test.go` |
| Address/parity frames trusted only for recently CRC-proven ICAO (dump1090 rule) | IMPL, UNIT | `report.go` `ProcessFrame`, `tracker.go` `KnownICAO`, `gate_test.go` |
| Beast TCP ingest with reconnect | IMPL, UNIT | `internal/radio/adsb/beast/beast.go` (Mode-AC `0x31` skipped; signal level and MLAT timestamp parsed then **discarded**) |
| Native 1090 PPM receiver | IMPL, UNIT | `internal/radio/adsb/ppm/receiver.go` — "fixed 2 Msps, magnitude-domain. Phase-corrected re-detection and 2.4 Msps operation are refinements left for later" (line 28) |
| CRC bit-error correction | **NOT-IMPL** | No single/two-bit fix anywhere in `internal/radio/adsb` (searched). readsb enables single-bit correction by default (external, readsb man page). |
| Overlapping-preamble handling | NOT-IMPL | `scan()` skips past each decoded frame (`i += preambleSamples + nbits*samplesPerBit`) |
| Global CPR (even/odd) | IMPL, UNIT (1 vector) | `cpr.go` `CPRDecodeGlobal`; one test pins the mode-s.org vector (52.2572°) |
| **CPR result range validation** | **DEFECT** | No `|lat| ≤ 90` check. A probe over 1 056 synthetic even/odd latitude pairs returned **542 `ok=true` results outside ±90°** (e.g. 125.996°). Matches the task's 222.798° observation. |
| **Surface position decode** | **DEFECT** | `tracker.go:118` sends `KindSurfacePosition` through the *airborne* `CPRDecodeGlobal`. Surface CPR uses a 90° zone and needs a reference position. The probe produced a plausible-looking but wrong 4.236°/−3.744° from a surface pair. Surface and airborne halves also share one per-ICAO slot, so they can be paired with each other. |
| Local (reference-based) CPR decode | NOT-IMPL | `cpr.go` header comment: "the obvious follow-up" |
| Receiver-range / speed plausibility gate | NOT-IMPL | No receiver lat/lon in the ADS-B path. readsb has `--max-range` (default 300 nm) and a speed check (external). |
| DF18 CF field (TIS-B / ADS-R / non-ICAO addresses) | **Risk, IMPL-gap** | `adsb.go:287` treats DF18 AA as an ICAO for every CF value. Non-ICAO/anonymous addresses can merge with real ICAO tracks. |
| **`/api/v1/adsb/aircraft/current` `crc_valid`** | **DEFECT** | `storage/aircraftlog.go:212` `CurrentAircraft` never SELECTs `crc_valid`, so the coalesced row's `CRCValid` is always `false`. This is exactly the task's "aggregated `crc_valid:false`". It is a reporting defect, not evidence of bad frames. |
| Coalesced position = last write wins, no plausibility | DEFECT (consequence) | Same function. One impossible position persists until the next position row. |
| Storage: one SQLite INSERT per Mode-S frame | IMPL — resource UNKNOWN | `aircraftlog.go` `insert`. At realistic message rates this is the dominant SD-card write and CPU cost (§6). |
| Decoder stats to `/metrics` | NOT-IMPL | `beast.Client.Stats()` / `ppm.Receiver.Stats()` exist but nothing in `cmd/` or `internal/api` reads them (searched) |
| Beast / SBS / aircraft.json **output** | NOT-IMPL | GT cannot be a data source for FR24, tar1090 or MLAT clients |
| Field (Pi) behaviour | **UNKNOWN** | Task: "Focused ADS-B Go tests reportedly passed in Docker; this is not evidence of field parity." Agreed. |

**The uncommitted Pi working-tree files** (`internal/radio/adsb/cpr.go` modified, `cpr.go.bak`, `cpr_invalid_test.go`, `Dockerfile.pi5-web`) are not in this repository. Their content is UNKNOWN. The CPR workstream (§10, ADSB-1) must start by retrieving them read-only (P-4). Do not re-derive.

**Architectural consequence:** under Option B, GT receives readsb's Beast stream and **re-decodes CPR itself**. That discards readsb's already range-checked, CPR-validated positions and replaces them with GT's. Two options, to be decided at ADSB-1:
- (a) fix GT's CPR to readsb parity (range check, surface decode, local decode, receiver reference), or
- (b) take positions from readsb's `aircraft.json` (or Beast plus readsb's JSON for positions) and use GT only for history, UI and correlation.

(b) is smaller and removes a whole class of divergence. (a) is needed anyway if native PPM is ever to replace readsb.

### 4.2 Flightradar24

| Item | Level | Evidence |
|---|---|---|
| GT acts as FR24 feeder | **NOT-IMPL** | No FR24 protocol, no Beast/SBS output (searched) |
| GT disturbs FR24 today | Low risk (IMPL reasoning) | GT is a TCP *client* of readsb's Beast port. It does not own the 1090 dongle. A GT crash or reconnect loop costs FR24 nothing. |
| FR24 health as acceptance metric | UNKNOWN | Read the fr24feed status page at :8754 before and after each stage (P-3) |

Preserve FR24 by **not changing its data path**: fr24feed ← readsb Beast (or its own `receiver=beast-tcp` config, P-3). Under no stage of this plan does GT sit between readsb and FR24.

### 4.3 DSC (VHF Ch-70, 156.525 MHz)

| Capability | Level | Evidence |
|---|---|---|
| FFSK 1300/2100 Hz demod, 1200 Bd | IMPL, UNIT | `internal/radio/dsc/ffsk/receiver.go` (comment labels 1300 Hz as "B"; per M.493 the higher tone is B. Labelling only — the dual-polarity hunt makes the sense moot.) |
| **10-bit character check** | **DEFECT (spec non-conformance)** | `dsc/bch.go`: CRC-3 with g(x)=x³+x+1. ITU-R M.493 §1.1.1 (Annex 1): bits 8–10 are the binary **count of B elements** in the seven information bits, information bits LSB first, check bits MSB first. A probe showed **17/128** M.493-valid characters pass GT's `BCHCheck`. The phasing DX character 125 is not among them (GT `1111101110` vs M.493 `1111101001`). |
| Phasing / DX lock | IMPL — depends on the wrong code | `dsc/receiver/receiver.go` locks on the DX phasing character via that syndrome check. On real air it should essentially never lock. |
| DX/RX time diversity | **NOT-IMPL** | `receiver.go` header: "read DX only, dropping the RX copies … a follow-up". M.493 §1.6.2 asks receivers to make maximum use of the signal (iterative decoding) |
| ECC (error-check character) | IMPL (field `EFCChecksumOK`) | Correctness UNKNOWN — validate with the code fix |
| Real-air validation | **NOT-IMPL** | `docs/decoder-capture-needs.md` lists DSC as capture-blocked ("confirm on-wire bit order, tone sense, and DX/RX offset"). `receiver.go` says the same. No `samples/dsc/`. |
| Alerting on DSC (distress/urgency) | **NOT-IMPL** | `internal/config/alerts.go:124` `AlertRuleEventKinds` has no `dsc.*` (nor AIS/ADS-B) kind |
| Field | **UNKNOWN — likely failing** | Check P-7: has `/api/v1/dsc/messages` on :8101 ever returned a row, and with what check status? |

Why the unit tests stay green: the FFSK modulator and the decoder share the same invented CRC-3. That is the self-consistent-synthetic trap this repository's own CLAUDE.md documents for SmartNet, YSF, FLEX, LoRa and NXDN. DSC is another member of that family.

### 4.4 AIS (161.975 / 162.025 MHz)

| Capability | Level | Evidence |
|---|---|---|
| GMSK → NRZI → HDLC → M.1371 parser | IMPL, UNIT | `internal/radio/ais/gmsk`, `ais/receiver`, `ais/ais.go` (types 1–27 enumerated; tests on gpsd AIVDM samples) |
| Front end: FM discriminator, single Mueller-Müller timing loop, zero-threshold slicer | IMPL | `gmsk/receiver.go` header. One demodulation hypothesis. AIS-catcher advertises multiple demodulation models (external, rtl-sdr.com). Relative sensitivity UNKNOWN. |
| Long (multi-slot) frames | IMPL (framer cap 1 024 B), UNIT for type 5 | `aprs/hdlc/framer.go:47`; `ais/receiver/receiver_test.go`. Note: `docs/ais.md` "multi-slot reassembly pending" describes a per-MMSI buffer. On air a multi-slot message is one longer HDLC frame, so the real open item is real-air validation, not reassembly. |
| **Both channels on one SDR** | **NOT-IMPL** | `cmd/gophertrunk/daemon.go:2797` builds one receiver per `ais.channels` entry. `runSingleChannelDecoder` (≈ line 5607) retunes the entry's dongle. Two entries on one serial would fight over the centre frequency. No validation rejects that (only `sdr.devices` duplicate serials are checked). |
| **AIVDM/NMEA output (UDP/TCP)** | **NOT-IMPL** | Searched `internal/radio/ais`, `cmd/gophertrunk`. Partner feeds cannot be served by GT. |
| **AIVDM/NMEA input (bridge from AIS-catcher)** | **NOT-IMPL** | No NMEA parser/listener. This is the main new code Option B needs. |
| CRC-failed frames published by default | IMPL (by design) | `ais/receiver` `DropBadFCS=false` publishes FCS-failed frames flagged `fcs_ok=false`. Views and exports must filter. |
| Real-air / field | **NOT-IMPL / UNKNOWN** | `docs/decoder-capture-needs.md`: "only the bit-stream synthetic is tested today". No `samples/ais/`. |

### 4.5 Cross-cutting platform

| Item | Level | Evidence / consequence |
|---|---|---|
| Event bus | IMPL | `internal/events/bus.go`: one bus, **64-event buffer per subscriber** (`daemon.go:944`), every subscriber sees every kind, a slow subscriber **drops** events. `Publish` returns a drop count that the Beast client and others ignore, so drops are **silent**. At ADS-B message rates a slow consumer (SQLite writer, SSE client, alert dispatcher) can drop **DSC events too**. DATA-INTEGRITY and SAFETY risk. |
| API auth | IMPL | `api.auth.mode: auto` (default) requires a bearer token for **mutations** on non-loopback binds and refuses them if no token is configured (`internal/api/auth.go:281`). **Read endpoints (positions, DSC, recordings) are unauthenticated.** |
| CORS | IMPL | Off by default (`config.go` `APICORSConfig`) |
| TLS | IMPL | `api.tls_cert` / `tls_key` optional |
| gRPC | IMPL | Exposed on :50051 in the image; review exposure |
| Health | IMPL | `/api/v1/health`. **DEFECT:** `docker-compose.yml:66` health check runs `wget`, but the runtime image installs only `ca-certificates` (`Dockerfile:29`). The check cannot run, so the container reports unhealthy (or the check is meaningless) under compose. Reproduce on the Pi with P-9. |
| USB | IMPL | Pure-Go RTL-SDR driver (`internal/sdr/rtlsdr/purego`, image built `CGO_ENABLED=0`). Serial pinning in config. Compose example maps a **bus/device path** (`/dev/bus/usb/003/002`), which changes on replug or reboot. Use the udev/serial approach in `docs/hardening.md`, or map `/dev/bus/usb` with device-cgroup rules. |
| Container hardening | IMPL | Non-root user, `cap_drop: ALL` + `DAC_OVERRIDE` in the example compose |
| Retention | IMPL | `internal/storage/retention.go` covers `aircraft_log`, `vessel_log`, `dsc_log`. The default for `retention.log_days` must be set deliberately. |
| Backup / restore | NOT-IMPL in-product | One SQLite file. Needs an operator procedure (§8.3). |

### 4.6 APRS and ACARS (future)

| Protocol | Level | Notes |
|---|---|---|
| APRS (Bell-202 AFSK → HDLC → AX.25 → APRS, Mic-E) | IMPL, UNIT | `internal/radio/aprs/*`. `decoder-capture-needs.md`: "only the synthetic test exists today". The #1316 note in CLAUDE.md fixed a full-rate front end that made every on-air frame `fcs_ok=false`. Still on-air-gated. Danish/EU frequency is 144.800 MHz. Needs its own dongle. |
| ACARS (AM MSK 2400 bps) | IMPL, UNIT, **REC-RF** | `docs/acars.md`: decodes the seven real-air messages in acarsdec's test recording and round-trips with acarsdec. **But** it runs on a `scanner.conventional` AM channel, so it listens on **one frequency at a time** (hops). acarsdec monitors several channels at once. For coverage, acarsdec plus a GT ingest is the stronger starting point. |

### 4.7 Summary maturity table

| Function | IMPL | UNIT | REC-RF | FIELD | Blocking defects |
|---|---|---|---|---|---|
| ADS-B via Beast ingest | ✔ | ✔ | (readsb is the RF decoder) | ✖ | CPR range, surface CPR, `crc_valid` aggregation, silent bus drops |
| ADS-B native PPM | ✔ | ✔ | ✖ | ✖ | no bit fix, 2 Msps magnitude-only, CPR |
| FR24 feed via GT | ✖ | — | — | — | not a feeder (by design keep fr24feed) |
| AIS native | ✔ | ✔ | ✖ | ✖ | 1 channel/SDR, no NMEA out |
| AIS ingest from AIS-catcher | ✖ | — | — | — | NMEA input to build |
| DSC native | ✔ | ✔ (self-consistent) | ✖ | ✖ (likely failing) | wrong check code, DX-only, no alerts |
| APRS native | ✔ | ✔ | ✖ | ✖ | real-air gate |
| ACARS native | ✔ | ✔ | ✔ (acarsdec sample) | ✖ | single channel at a time |
| Unified UI/API/store | ✔ | ✔ | n/a | ✖ | read auth, integrity filters, metrics |

---

## 5. Safety posture (DSC)

1. **GopherTrunk DSC is not GMDSS watch-keeping and must never be presented as such.** It is a hobby receiver on a single RTL-SDR, with no guaranteed availability, a decoder not yet conformant with M.493, and no alert path. The UI, API docs and any alert text must say "informational, not a safety-of-life service". The Pi must not replace a type-approved DSC radio on the boat.
2. Until DSC-1..3 (§10) pass, DSC records are **unverified**. Show them with check status and never forward them to third parties.
3. Measure alert latency (§6) from burst end to event, store and UI/alert, with a stated bound. Do not promise delivery: the bus is drop-on-slow (§4.5).
4. DSC events need a priority path that cannot be starved by ADS-B volume (PLAT-1).

---

## 6. Quantitative acceptance plan

**Principle:** compare like with like. Only these comparisons count as **comparable**:
- (i) the same IQ recording replayed through both decoders, or
- (ii) simultaneous decoding of the same antenna through a splitter, accounting for the splitter loss.

Anything else (different times, antennas or dongles) must be flagged **NON-COMPARABLE**. It can only show "works / doesn't work".

### 6.1 Windows and bookkeeping
- Matched windows of **60 min**, repeated at ≥ 3 times of day (traffic varies). Record start/stop UTC, gain, ppm, antenna and filter/LNA chain for each.
- All counts come from the decoder's own counters and from the persisted store, and the two must reconcile. A mismatch between the source counter and GT rows is a data-integrity failure in itself.

### 6.2 Metrics by service

| Service | Metric | Pass criterion (proposed — operator to confirm) |
|---|---|---|
| ADS-B (ingest, B) | Beast frames read vs GT `aircraft_log` rows | ≥ 99.9 % of CRC-valid DF11/17/18 frames persisted. Bus drop counter = 0. |
| | Unique ICAO per window: readsb `aircraft.json` vs GT `current` | GT ⊇ readsb for aircraft with ≥ 2 messages |
| | Positions outside ±90° / ±180°, or > `max_range` from receiver | **0** |
| | Position agreement with readsb for the same ICAO/time | ≤ 0.01° (≈ 1 km) for 99 %; surface positions either correct (with reference) or suppressed |
| | `crc_valid` on `current` | true when the contributing frames were CRC-valid |
| | Data freshness (readsb → GT API) | p95 ≤ 2 s |
| ADS-B (native, C gate) | Same-RF: DF17 count, unique ICAO, positions per aircraft vs readsb | ≥ 95 % of readsb on the same recording, 0 invalid positions — **not attempted until ADSB-4** |
| FR24 | fr24feed status (:8754): connected, aircraft count, MLAT state | Unchanged ± normal variance across every stage |
| AIS (ingest, B) | NMEA sentences from AIS-catcher vs GT rows (by type) | ≥ 99.9 % persisted; multipart sentences reassembled; checksum-failed sentences rejected and counted |
| | Unique MMSI per window | GT = AIS-catcher |
| | Channel A/B split | Present and plausible (both channels non-zero) |
| AIS (native, C gate) | Same-IQ replay: CRC-valid messages per channel, unique MMSI, message-type mix vs AIS-catcher | ≥ 95 % of AIS-catcher, both channels from a single dongle — requires AIS-N1/N2 first |
| DSC | Known-good recording(s): sequences decoded, format/category/MMSI/nature/position fields | 100 % field match on clean bursts; per-character DX/RX usage reported |
| | False-positive rate on ≥ 24 h of idle Ch-70 | 0 decoded sequences with ECC failure published as valid |
| | Alert latency (burst end → API row / alert delivered) | p95 ≤ 5 s; drops = 0 under ADS-B load |
| All | CPU % per container, RSS, SQLite DB growth/day, SD writes/day (`iostat`/`vmstat`) | Pi 5 total CPU ≤ 60 % sustained, no throttling (`vcgencmd get_throttled` = 0x0), DB growth within retention budget |
| All | Uptime / restarts / USB disconnects (`docker events`, `dmesg`) | 0 unexplained restarts per 7 days; reconnect without manual action |
| All | Signal quality fields preserved | Beast signal level, AIS-catcher signal/PPM (if offered in NMEA+/JSON) available in GT, or explicitly declared dropped |

---

## 7. Unknowns and read-only Pi checks (no restarts, no writes)

Run these and paste the outputs into the PR before Phase 2 is authorised. All commands are read-only.

| ID | Closes | Command(s) |
|---|---|---|
| P-1 | Runtime (Docker vs LXD), container inventory, restart policy | `docker ps --format '{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'`; `docker inspect <c> --format '{{.HostConfig.RestartPolicy}} {{.HostConfig.Devices}} {{.HostConfig.Privileged}}'`; `lxc list 2>/dev/null` |
| P-2 | SDR inventory, serials, tuners, USB topology, bus contention | `lsusb -t`; `for d in /sys/bus/usb/devices/*; do [ -f $d/serial ] && echo $d $(cat $d/idVendor):$(cat $d/idProduct) $(cat $d/serial); done`; `dmesg | grep -iE 'usb|rtl|over-current' | tail -50` |
| P-3 | FR24 data path | `curl -s localhost:8754/monitor.json`; read fr24feed `receiver=`/`host=` config (inside its container, read-only `cat`) |
| P-4 | Uncommitted CPR work | In `~/gophertrunk-eval`: `git status --porcelain; git diff -- internal/radio/adsb/cpr.go; cat internal/radio/adsb/cpr_invalid_test.go Dockerfile.pi5-web; git rev-parse HEAD` |
| P-5 | GT ADS-B symptom | `curl -s localhost:8101/api/v1/adsb/aircraft/current \| jq '[.[] \| select(.latitude>90 or .latitude<-90)] \| length'`; `curl -s 'localhost:8101/api/v1/adsb/aircraft?limit=2000' \| jq 'group_by(.kind) \| map({k:.[0].kind,n:length,bad:[.[]\|select(.has_position and (.latitude\|fabs)>90)]\|length})'` |
| P-6 | readsb reference counts | `curl -s localhost:8080/data/aircraft.json \| jq '.aircraft\|length'`; `curl -s localhost:8080/data/stats.json \| jq .last1min` |
| P-7 | Is GT DSC decoding anything? | `curl -s 'localhost:8101/api/v1/dsc/messages?limit=50' \| jq 'length, map(.fcs_ok // .ecc_ok)'`; GT logs: `docker logs --since 24h <gt> 2>&1 \| grep -iE 'dsc' \| tail -50` |
| P-8 | AIS-catcher outputs and partners | AIS-catcher config/command line (`docker inspect` Args/Env); `ss -uapn \| grep -i ais`; confirm the possible duplicate feed from digipi-2 |
| P-9 | Health check defect | `docker inspect <gt> --format '{{json .State.Health}}'` |
| P-10 | Resources baseline | `uptime; vcgencmd get_throttled; vcgencmd measure_temp; free -m; docker stats --no-stream; df -h; iostat -x 5 3` |
| P-11 | GT exposure | GT config `api:` block (read-only); `ss -tlpn \| grep -E '8101\|50051\|8080'` |
| P-12 | SQLite load | `ls -la <calls.db>*`; `sqlite3 -readonly <db> 'select count(*) from aircraft_log where received_at > strftime("%s","now","-1 hour")*1e9'` |

Unknowns that need RF, not shell:
- **U-RF1:** a ≥ 60 s IQ recording at 1090 MHz, 2.4 MS/s (for the native-vs-readsb comparison, later).
- **U-RF2:** a dual-channel AIS IQ recording (162 MHz centre, ≥ 288 kS/s) from the AIS dongle while AIS-catcher is paused. That needs an operator-approved maintenance window, so it is *not* part of Phase 1.
- **U-RF3:** one or more **real DSC bursts** (IQ, 156.525 MHz). A test call from the operator's own type-approved VHF on Ch-70 (routine individual call to own MMSI / test call per local rules) is the cleanest ground truth. Record IQ with `gophertrunk capture` on SDR C.

---

## 8. Staged migration, rollback, configuration and security

Each stage has an entry gate, an exit gate and a rollback. **Every stage needs explicit operator approval.** No stage touches readsb's, FR24's or AIS-catcher's dongle or configuration.

| Stage | Change | Entry gate | Exit gate (§6 metrics) | Rollback |
|---|---|---|---|---|
| S0 | Baseline: run P-1..P-12; snapshot configs and DB | Phase 1 approved | Baseline numbers recorded | n/a |
| S1 | Ship ADS-B integrity fixes (ADSB-1..3, PLAT-1/2) to GT. GT stays a Beast client only. | Fixes merged with failing-first tests | 0 invalid positions; `crc_valid` correct; drop counter 0 for 7 days; FR24 unchanged | Re-deploy the previous GT image tag; GT is a pure consumer |
| S2 | AIS NMEA ingest: AIS-catcher adds one extra UDP/TCP output to GT (the partner outputs are untouched) | AIS-B1 merged | MMSI parity for 7 days; partner feeds unchanged | Remove the extra AIS-catcher output line |
| S3 | DSC fix (DSC-1..3) on the existing GT DSC dongle | U-RF3 recording decodes 100 % | False-positive and latency gates; safety wording in UI | Previous image; DSC is informational only |
| S4 | Unified UI behind the existing nginx :80 (keep the tar1090 and AIS-catcher UIs reachable) | S1–S3 green | Operator acceptance | nginx route revert |
| S5+ | New services (APRS, ACARS) on **new** dongles, one at a time | Per-protocol REC-RF gate | Per-protocol | Disable the entry |
| Native replacement of readsb or AIS-catcher | **Not planned.** Only reconsidered if ADSB-4 / AIS-N1..N3 meet the same-RF gates in §6 | — | — | Keep the specialist container installed and stopped, never deleted, for ≥ 30 days |

### 8.1 Configuration and security baseline (target)
- GT `api.http_addr` on LAN only, or `127.0.0.1` behind nginx. `api.auth.mode: required` with `token_file` (a mounted secret, not inline in YAML).
- **Read endpoints:** either restrict to LAN at nginx, or add read auth (PLAT-4). Positions and DSC traffic should not be publicly exposed.
- gRPC (`api.grpc_addr`, :50051): bind to loopback unless used. `rigctld` off.
- TLS at nginx if accessed beyond the LAN. Prefer VPN over port-forwarding.
- Pin images by digest. Keep the previous tag available for rollback.
- USB: map by udev rule and serial, not by `/dev/bus/usb/BBB/DDD`. One container per dongle owner.
- Do not run any SDR container `--privileged`.

### 8.2 Observability
Export per-source counters to `/metrics` (PLAT-2): Beast frames read, decoded and dropped; bus drops per subscriber; AIS and DSC messages, valid and invalid; last-message age per source. Alert on staleness (e.g. no ADS-B rows for 5 min while readsb reports traffic).

### 8.3 Persistence, backup and restore
- Volumes: GT DB, recordings and config on the host. Back up the DB with `sqlite3 <db> ".backup <file>"` (consistent online copy) daily to off-SD storage, then do a restore drill once before S1.
- Retention: set `retention.log_days` explicitly for `aircraft_log`. Consider not persisting every Mode-S frame (ADSB-5).

---

## 9. Source ownership

- **Upstream:** `MattCheramie/GopherTrunk` (Go module path). **Fork:** `froekjaer/GopherTrunk`. The upstream moves very fast (hundreds of PRs). Carry local patches as small, upstreamable PRs. ADS-B CPR, the `crc_valid` aggregation, DSC M.493 conformance and the health-check fix are all upstream-worthy bugs.
- Tracking rule: every local change gets an upstream issue or PR reference. Rebase the fork onto upstream tags, not onto `main` HEAD.
- Specialist software (readsb/ultrafeeder, fr24feed, AIS-catcher) stays on its upstream images, pinned by digest.
- The Pi working tree `~/gophertrunk-eval` is a third, unversioned source of truth. Bring it under version control (P-4) before any CPR implementation.

---

## 10. Prioritised, dependency-aware backlog

Each item follows CLAUDE.md's bug-fix discipline: a regression test that **fails without the fix**, and where possible pinned to an *independent* reference (literal vectors, a recording) rather than GT's own encoder.

| ID | Item | Depends on | Acceptance criteria |
|---|---|---|---|
| **ADSB-0** | Retrieve the Pi's uncommitted CPR work (P-4) and commit it to a branch as-is | P-4 | Files preserved byte-for-byte in git |
| **ADSB-1** | CPR correctness: reject `|lat|>90`, separate airborne/surface state, surface decode only with a reference position (otherwise suppress), local decode against the receiver position or last good position, optional max-range gate | ADSB-0 | Literal vectors from mode-s.org / pyModeS for airborne, surface and zone-boundary cases. Probe-style property test: 0 out-of-range results from random pairs. Field: 0 invalid positions over 7 days. |
| **ADSB-2** | `CurrentAircraft` returns real `crc_valid` (and never coalesces a position from an invalid row) | — | Failing-first test in `storage/aircraftcurrent_test.go` |
| **ADSB-3** | DF18 CF handling (do not merge non-ICAO addresses into ICAO tracks) | — | Literal DF18 CF≠0 vectors |
| **PLAT-1** | Bus: count and export drops; separate or priority subscription for DSC; larger buffer for the high-rate ADS-B writer | — | Load test: 2 000 aircraft events/s with DSC events interleaved → 0 DSC drops |
| **PLAT-2** | `/metrics` for Beast, PPM, AIS, DSC and storage counters, plus last-seen age | — | Counters reconcile with DB rows (§6.1) |
| **PLAT-3** | Fix the compose health check (use `gophertrunk` itself or install `wget`) | — | `docker inspect` health = healthy |
| **ADSB-5** | Storage load: batch inserts or persist the aggregated state instead of every frame (configurable) | PLAT-2 | Measured CPU/IO on the Pi at the P-6 message rate is within budget |
| **AIS-B1** | NMEA/AIVDM ingest (UDP + TCP), multipart reassembly, checksum validation, channel A/B tag, source tagging | — | gpsd AIVDM corpus; parity with AIS-catcher over 7 days (§6) |
| **DSC-1** | Replace CRC-3 with the M.493 zero-count 10-bit code, with correct bit order (info LSB-first, check MSB-first) | — | Literal table vectors from M.493 Table A1-1. The synthetic modulator moved to the same code from an *independent* encoder. |
| **DSC-2** | DX/RX time-diversity merge plus ECC validation | DSC-1 | Bursts with injected single-copy errors recover. Real-air recording decodes. |
| **DSC-3** | Real-air validation on U-RF3 plus safety wording; add `dsc.*` to alert kinds with distress/urgency rules | DSC-1/2, PLAT-1 | 100 % field match on the recording; latency gate |
| **PLAT-4** | Optional read-auth for position/DSC endpoints | — | Unauthenticated read → 401 when enabled |
| **AIS-N1** (later) | Native single-dongle dual-channel AIS (one tuner centred at 162.000 MHz, two DDC taps) | AIS-B1 done | Same-IQ ≥ 95 % of AIS-catcher |
| **AIS-N2** (later) | NMEA output so GT could serve partners | AIS-N1 | Partner accepts feed in a parallel trial |
| **ADSB-4** (later) | Native PPM parity: 2.4 MS/s, phase correction, 1-bit CRC fix, overlapping preambles | ADSB-1 | Same-IQ ≥ 95 % of readsb (§6) |
| **APRS-1 / ACARS-1** (later) | Real-air gates; ACARS multi-channel, or acarsdec ingest | new dongles | Per decoder-capture-needs |

### 10.1 Top three smallest safe next implementation steps
1. **ADSB-2** — one SQL column and one test. It fixes a misleading API field with zero RF risk.
2. **ADSB-1 (range-check slice only)** — reject `|lat|>90` / `|lon|>180` and stop routing surface positions through airborne CPR (suppress them until the reference decode exists). It removes the 222.798° class of error. Full surface/local decode follows in the same workstream.
3. **PLAT-1 (drop counting only)** — surface `Publish` drop counts as a metric and a log line. It turns a silent integrity failure into a visible one before any load increases.

DSC-1 is the most valuable fix but is not "smallest safe". It changes the decoder's on-air behaviour and must land together with a real-air recording (U-RF3).

### 10.2 Revised Codex task breakdown (each a separate, explicitly authorised task)
- **T1** ADSB-0 + ADSB-2 + ADSB-1 range slice — one PR, `Refs` the CPR issue, failing-first tests.
- **T2** ADSB-1 full (surface + local + receiver reference) + ADSB-3.
- **T3** PLAT-1 + PLAT-2 + PLAT-3.
- **T4** AIS-B1 (NMEA ingest), plus a docs correction to `docs/ais.md` "multi-slot reassembly".
- **T5** DSC-1 + DSC-2, gated on U-RF3 being available; DSC-3 after.
- **T6** Deployment: compose/stack for the S1–S4 topology, digest pinning, backup job. Operator-applied, not Codex-applied.

---

## 11. What was actually run in this assessment

All on the cloud workspace (x86-64, 2 vCPU, Go 1.26.11), **not on the Pi**:

```
go test ./internal/radio/adsb/... ./internal/radio/ais/... ./internal/radio/dsc/... \
        ./internal/radio/aprs/... ./internal/radio/acars/... ./internal/storage/
```
All 17 packages passed (`ok`). As explained above, a pass says nothing about real-air correctness.

Two **throw-away probe tests** were written, run, and deleted (not committed):
1. `adsb`: swept 32 × 33 even/odd CPR latitude pairs through `CPRDecodeGlobal`. **542** returned `ok=true` with |lat| > 90 (example: 125.996°). Fed a surface even/odd pair through `Tracker.Update`. It returned a "global" position computed with the airborne formula.
2. `dsc`: compared GT's `BCHEncode`/`BCHCheck` against the M.493 zero-count code for all 128 symbols. **17/128** agree. Phasing character 125: GT `1111101110`, M.493 `1111101001`.

No benchmarks were run. **No CPU or RAM figures in this document are measurements.** All resource statements are marked UNKNOWN and assigned to P-10/P-12.

---

## 12. Challenge to the proposed architecture (counterarguments considered)

- *"Option A is good enough."* For reception and feeds it is. It loses only the unified history and UI. If the GT fixes in §10 are not resourced, staying on A is the correct, safer outcome. The plan is designed so that stopping after S0 leaves A intact.
- *"Make GT the RF owner too, since it already decodes everything."* The repository's own history (CLAUDE.md) shows many decoders that were green on synthetic tests and wrong on air. DSC is shown here to be one of them, and ADS-B CPR has a live symptom. Ownership should follow field evidence, and none exists yet for GT's 1090/162/156 MHz RF paths.
- *"Re-decoding Beast in GT gives one consistent decoder."* It gives a *second* decoder whose output diverges from what FR24 receives from readsb. Two maps showing different positions for one aircraft is worse than one. Prefer readsb positions (ADSB-1 option b) unless and until GT reaches parity.
- *"One process is simpler to operate."* It is simpler to deploy but has a larger blast radius. The bus shares one drop-on-slow channel per subscriber across all protocols. Separation is a real resilience property here, not an aesthetic choice.
- *Counter-counter:* a single store and UI does have real value — cross-protocol correlation (e.g. DSC MMSI ↔ AIS vessel), one retention and backup policy, one alerting system. That is why B is recommended over A, *once* integrity is fixed.

---

## 13. Open questions for the operator

1. Docker or LXD/LXC on the Pi today (earlier notes say LXC)? Which tuner is in each dongle?
2. Which AIS partner feeds exist (earlier notes say four UDP shares), and is the digipi-2 duplicate still live?
3. Is FR24 fed from ultrafeeder's Beast port or from its own receiver config?
4. What should GT's unified view be for — personal situational awareness, history/analytics, or anything shared with third parties? This decides read-auth and retention.
5. Can you make a Ch-70 DSC test call (own radio, per local rules) while SDR C records IQ? That is the single most useful piece of evidence for this plan.
6. Will you approve a maintenance window to pause AIS-catcher for an IQ recording (U-RF2)? Only needed if native AIS is ever pursued.
7. Acceptable SD-card write budget, or is there an SSD/USB disk for the DB?

---

### References (external, not re-verified on the Pi)
- ITU-R M.493 (all versions -8 to -16), Annex 1 §1.1.1: ten-bit error-detecting code; bits 8–10 = count of B elements; info LSB first, check MSB first. https://www.itu.int/rec/R-REC-M.493
- readsb man page: `--fix` single-bit correction (default), `--max-range` (default 300 nm), Beast out port 30005. https://manpages.ubuntu.com/manpages/plucky/man1/readsb.1.html
- AIS-catcher: dual-channel receiver for RTL-SDR, NMEA over UDP/TCP/HTTP, multiple decoding models. https://github.com/jvde-github/AIS-catcher , https://www.rtl-sdr.com/ais-catcher-a-dual-band-multiplatform-ais-receiver-for-rtl-sdr-and-airspy-hf-with-multiple-decoding-models/
