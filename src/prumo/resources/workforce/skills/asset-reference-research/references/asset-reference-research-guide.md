# Asset Reference Research — Technical Reference Guide

## 1. Core Concepts

### 1.1 Briefs Before Browsing
Research starts from a frozen brief, not a search box. One brief row fully determines what counts as a hit:

```md
- Need A7: 2D platformer hero run cycle, 8 frames, 512 px sprites,
  pivot at feet center, style anchor: Kenney platformer art deluxe,
  fills placeholder greybox_hero_run in scenes 2 and 5.
```

Vague needs produce vague candidates; return them for budgets and slots before spending sweep time.

### 1.2 Trusted Sources and Quarantine
Sweep in reliability order: Kenney.nl for CC0 packs with consistent style, OpenGameArt for breadth with per-file license variance, Poly Pizza for CC0 low-poly 3D, Unsplash for CC0 photography textures, Freesound for CC0 and CC-BY audio with per-sample terms. Aggregator re-uploads and search-engine images are quarantined until the original author page and license are found — screenshots of licenses are not proof.

### 1.3 The License Matrix
Every candidate lands in exactly one class. Approved for commercial builds: CC0 1.0 and CC-BY 4.0 with attribution shipped. Conditional: CC-BY-SA 4.0 only after confirming share-alike does not attach to proprietary code or locked game content, and OFL 1.1 for fonts with reserved-name checks. Forbidden commercially: CC-NC variants, CC-ND variants, unknown or custom uploader text, and ripped commercial-game content regardless of claims.

### 1.4 Attribution That Survives Shipment
A shippable entry names everything a re-verifier needs:

```md
- Starfield Parallax Layer 2 by Kenney Vleugels, Kenney.nl pack
  space-shooter-redux, CC0 1.0, cropped from 1024 to 512 px wide,
  archive sha256:4ad1c8f2e9b047aa91d6c3f58b2e77d0c1a9456be708f3d2aa51b6c9d07e1234.
```

Store the source archive beside the hash under `assets/vendor/` so any future contributor can re-derive the proof.

## 2. Technical Fit Checks
- Sprites: pivot, frame size, and tile grid must match the placeholder or the brief is resized first.
- Audio: loop points verified in-editor, sample rate unified to 48 kHz before import.
- Fonts: glyph coverage checked for pt-BR diacritics and en-US sets; missing glyphs fail the mapping.
- 3D: triangle counts and rig bone names must fit the importer profile, for example 5k triangles for background props.

## 3. Common Pitfalls
- Treating editorial or NC-licensed photography as game-safe because it previewed nicely.
- Shipping attribution with dead source URLs and no archived copy, which voids re-verification.
- Mixing share-alike assets into proprietary bundles without a compatibility ruling on file.
