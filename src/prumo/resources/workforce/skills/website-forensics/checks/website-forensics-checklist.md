# Website Forensics — Verification Checklist

## 1. Scope & Conduct
- [ ] At most 5 forensic questions defined before inspection; no open-ended browsing.
- [ ] Public pages only; robots.txt honored; throttled identified requests; 429/captcha stops work.
- [ ] No paywall/auth/anti-bot bypass; no bulk scraping of content, media, or user data.

## 2. Layout Measurements
- [ ] Container max-widths, grid columns, and gutters recorded at 360/768/1280/1920 px viewports.
- [ ] Header/footer anatomy and above-the-fold hierarchy documented with captures.
- [ ] Breakpoints where layout collapses identified explicitly.

## 3. Typography & Color Tokens
- [ ] Type roles recorded (family, weight, size, line-height) with font-loading strategy noted.
- [ ] Color roles sampled as hex values with computed contrast ratios per pair.
- [ ] Findings mapped onto our token scale with adopt/diverge decisions.

## 4. Interaction & State Coverage
- [ ] Primary-flow states captured: default, hover, focus, loading, empty, error.
- [ ] Missing focus indicators or color-only signaling flagged as cautions, not patterns.
- [ ] Failing (sub-AA) patterns observed are marked do-not-copy with rationale.

## 5. Deliverable & Evidence
- [ ] Teardown answers each scoped question with measurements and annotated captures.
- [ ] Deliverable contains patterns and measurements only — no redistributed assets.
- [ ] `scripts/verify.sh` exits 0 from the repository root.
