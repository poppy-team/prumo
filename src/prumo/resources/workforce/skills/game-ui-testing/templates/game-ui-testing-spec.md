# Game UI Test Specification — Starfall Drift v0.9.2

## 1. Build and Scope Under Test
- **Game and build**: Starfall Drift, build 0.9.2-rc3 (Windows 64-bit, Steam Deck profile)
- **Screens inventoried**: Title, Main Menu, Ship Select, Options (Audio/Video/Controls tabs), Pause, Race Results, Store Bundle Dialog — 9 screens total
- **Input devices**: Xbox Wireless Controller (firmware 5.23), keyboard (WASD + arrows + Enter/Esc), touch swipe on Steam Deck touchscreen
- **Test date and owner**: 2026-09-10, UI QA pod (Mara Voss), engine Godot 4.3

## 2. Resolutions and Safe Areas

| Target | Resolution | Action-safe inset | Title-safe inset | Verdict |
|---|---|---|---|---|
| Living-room TV | 1920x1080 | 67 px | 96 px | Pass, overlay screenshots archived |
| Handheld LCD | 1280x720 | 45 px | 64 px | Pass, HUD fuel bar re-anchored II-118 |
| 4K monitor | 3840x2160 | 134 px | 192 px | Pass |
| Ultrawide | 3440x1440 | 120 px | 172 px | Pass with pillarboxed cinematics |

## 3. Focus Navigation Results

| Screen | Initial focus | Stops visited | Traps found | Verdict |
|---|---|---|---|---|
| Main Menu | Race button | 6 of 6 | 0 | Pass |
| Ship Select | Falcon S-11 card | 11 of 11 | 0 | Pass |
| Options / Controls | Reset Defaults | 14 of 14 | 1 fixed (slider row dropped focus, fixed in commit 8f2c41) | Pass after fix |
| Pause | Resume | 5 of 5 | 0 | Pass |
| Store Bundle Dialog | Confirm Purchase | 3 of 3 | 0 | Pass, debounce 250 ms verified |

## 4. Localization Results

| Locale | Expansion observed | Overflow defects | Missing glyphs | RTL check | Verdict |
|---|---|---|---|---|---|
| Pseudo (140%) | up to 41% on Ship Select | 0 after container fixes | n/a | n/a | Pass |
| pt-BR | 34% on Race Results | 1 fixed (podium label wrapped, commit 91ad02) | 0 | n/a | Pass after fix |
| ja-JP | line breaks per kinsoku | 0 | 0, Noto Sans JP fallback active | n/a | Pass |
| ar-SA | mirrored nav bar, LTR numerals kept | 0 | 0 | Pass, progress direction mirrored | Pass |

## 5. State Transitions Under Load (50 ms Clamped Frame Time)

| Transition | Median latency | Max observed | Double activation | Verdict |
|---|---|---|---|---|
| Pause during race | 180 ms | 340 ms | none | Pass |
| Resume to countdown | 220 ms | 410 ms | none | Pass |
| Store dialog confirm | 140 ms | 260 ms | none, 250 ms lockout held | Pass |
| Scene change Menu to Ship Select | 310 ms | 480 ms | none | Pass |

## 6. Regression Evidence
- [x] Focus-sweep GUT suite: 9 screens, 400 moves each, zero homeless-focus frames on rc3.
- [x] Screenshot baselines: 9 screens x 4 resolutions x 4 locales = 144 PNGs, pixel diff at or below 0.4%.
- [x] `scripts/verify.sh` exits 0 on the skill package; game CI gate `ui-regression` green on commit 91ad02.
