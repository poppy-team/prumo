# Reality Verification Report — Atomic Upload Replacement

## 1. Claim Under Test
- **Claim**: "Upload handler atomically replaces files with crash-safe semantics."
- **Date**: 2026-09-17
- **Author**: storage-team
- **Scope**: `api/upload.py:112`, `fs/atomic_write.py`, CLI flag `--no-atomic`

## 2. Terminal State Assessment
- **Implemented**: Yes — `fs/atomic_write.py` (78 lines) present.
- **Reachable**: Yes — call path `POST /upload → api/upload.py:112 → atomic_write()` traced; `--no-atomic` bypass documented as accepted dev-only path (flag hidden from `--help`).
- **Exercised**: Partially — existing `test_atomic_replace` uses in-memory fake FS (does not execute `os.replace`); error path (disk full) uncovered.
- **Evidenced**: After fix — `test_atomic_replace_real_tmp` (real `tmp_path`) + `test_atomic_enospc` (full-volume fixture) recorded below.
- **Verified**: Yes — mutation kill confirmed (see §4). **Accepted**: pending storage-team sign-off. **Released**: no.

## 3. Call-Site Inventory
- `api/upload.py:112` (production path, authenticated uploads)
- `cli/admin.py:44` (admin re-upload, same function)
- Error path: `OSError` → RFC 7807 507 with `upload_id` correlation (verified in test)
- Dead code found: `fs/atomic_write_legacy.py` — zero inbound references; scheduled for deletion (cleanup task C-118)

## 4. False-Green Hunt
- Mutation 1: removed `f.flush()+os.fsync` → `test_atomic_replace_real_tmp` FAILS (kill confirmed).
- Mutation 2: replaced `os.replace` with plain `open().write()` → ENOSPC test FAILS (kill confirmed).
- Mock audit: fake-FS test retained but relabeled `unit`; real-FS tests labeled `integration` and gated in CI.

## 5. Evidence & Next Steps
- `pytest tests/integration/test_atomic_upload.py` 14/14 green; mutation kills 2/2.
- Next: storage-team acceptance review, then delete `atomic_write_legacy.py` (C-118).
