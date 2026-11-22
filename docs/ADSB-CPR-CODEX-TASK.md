# ADS-B CPR implementation task for Codex

Work only on branch `fix/adsb-cpr-position-decoding`. Do not modify main, merge, deploy, restart containers, or change production configuration.

## Objective
Implement standards-correct ADS-B airborne global CPR, local CPR, and surface CPR for Beast and native PPM receivers.

## Findings to verify in source
- `internal/radio/adsb/tracker.go` handles airborne and surface with the same global airborne CPR routine, potentially causing invalid surface positions (a live report contained latitude 222.798).
- Beast and PPM instantiate separate trackers in `internal/radio/adsb/beast/beast.go` and `internal/radio/adsb/ppm/receiver.go`.
- `internal/config/config_peripherals.go` defines `ADSBBeastConfig` and `ADSBChannelConfig`, with no receiver location fields.
- `cmd/gophertrunk/daemon.go` wires up both receivers.
- `internal/config/config_validate.go` has no ADS-B-specific section validator.
- Aggregated aircraft API reported `crc_valid:false`; investigate separately and test any confirmed defect.

## Acceptance criteria
1. Correct global airborne CPR, local airborne CPR with bounded trusted reference, and 90-degree surface CPR disambiguated by trusted reference. Reject ambiguous coordinates.
2. Separate airborne/surface even-odd state per ICAO; protect against stale/out-of-order messages and implausible position jumps without discarding valid fast aircraft.
3. Optional, individually configured receiver coordinates for Beast upstreams and PPM channels; robust validation of absent/partial/out-of-range/nonfinite values; backward compatibility.
4. Preserve CRC verification, event publishing, API behavior and existing tests.
5. Add independently sourced CPR vectors, boundary, transition, source isolation, configuration and regression tests. Compare identical Mode-S frames with readsb when feasible.
6. Report tests run, uncertainties, changed files and safe deployment/rollback steps. Open draft PR, not merge.

## Pi-specific safety
The Raspberry Pi checkout `~/gophertrunk-eval` contains uncommitted `internal/radio/adsb/cpr.go` and untracked `Dockerfile.pi5-web`, `internal/radio/adsb/cpr.go.bak`, `internal/radio/adsb/cpr_invalid_test.go`. These are **not automatically present in GitHub**. Preserve and reconcile before any deployment; never overwrite them.

## Execution
Inspect source and tests first. Implement on this branch using Codex, run Go tests, and submit a draft PR for review. This task document is not evidence that Codex has been started.
