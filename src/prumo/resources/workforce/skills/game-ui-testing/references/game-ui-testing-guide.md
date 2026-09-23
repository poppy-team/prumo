# Game UI Testing Reference Guide

## 1. Core Concepts

### 1.1 Focus Graph
Game UI navigation is a directed graph: nodes are focusable widgets, edges are directional moves (up/down/left/right). A correct focus graph is strongly connected over its screen (every node reachable) with no sink that drops focus to void. Engines expose this explicitly: Godot `focus_neighbor_top/bottom/left/right` and `focus_next/previous`, Unity `Selectable.navigation` (Automatic, Explicit, None), Unreal `FNavigationConfig` with `NavigationRebuild`.

### 1.2 Safe Areas
Overscan on TVs and notches on handhelds clip screen edges. Two standard rectangles apply:
- **Action-safe (3.5% inset)**: all interactive controls must sit inside.
- **Title-safe (5% inset)**: all critical text and HUD readouts must sit inside.
Inset margin in pixels: `margin_px = 0.035 * min(width, height)` for action-safe. At 1920x1080 the action-safe rectangle is 1852x1012 centered.

### 1.3 Anchors, Containers, Nine-Patch
Resolution independence comes from relative layout: anchors pin widget edges to parent fractions, containers (HBox/VBox/Grid) auto-arrange children, and nine-patch (9-slice) textures scale borders without stretching corners. Any absolute pixel offset tied to rendered text size is a localization defect waiting to happen.

### 1.4 Text Metrics and Expansion
Translated strings expand: German and Brazilian Portuguese run 30-40% longer than English source; CJK scripts need full glyph coverage and line-breaking rules (kinsoku); Arabic and Hebrew need RTL mirroring plus bidirectional text handling. Pseudo-localization simulates expansion before translators deliver: wrap each string as `[!!! Şţŕîñĝ !!!]` padded to 140% length.

## 2. Patterns and Anti-Patterns

| Pattern (do) | Anti-pattern (do not) |
|---|---|
| Declare explicit focus neighbors per screen; auto-generate sweep tests from them | Rely on engine "automatic" navigation and hope testers find traps |
| Draw safe-area overlay in debug builds; gate screenshots on it | Eyeball margins on one monitor and ship |
| Use positional placeholders: `Defeated {0} in {1} turns` | Concatenate fragments: `"Defeated " + name + " in " + n + " turns"` |
| Debounce confirm at the state machine (250 ms lockout) | Debounce in each button handler, or not at all |
| One screen owns input; modal stack pauses lower layers | Broadcast input to all visible layers and let z-order decide |

## 3. Minimal Example: Focus Sweep Test (Godot GUT)

```gdscript
# tests/test_main_menu_focus.gd
extends GutTest

const MENU_SCENE := "res://ui/main_menu.tscn"
const MOVES := ["ui_up", "ui_down", "ui_left", "ui_right", "ui_accept"]

func test_focus_never_homeless_during_sweep():
    var menu := load(MENU_SCENE).instantiate()
    add_child(menu)
    await get_tree().process_frame
    var seen := {}
    for i in range(400):
        var owner: Control = menu.get_focus_owner()
        assert_not_null(owner, "Focus lost to void on step %d" % i)
        seen[owner.name] = true
        Input.action_press(MOVES[i % MOVES.size()])
        await get_tree().process_frame
        Input.action_release(MOVES[i % MOVES.size()])
        await get_tree().process_frame
    assert_gte(seen.size(), 5, "Sweep visited fewer controls than the menu declares")
    menu.queue_free()
```

The test loads the real menu scene, performs 400 directional moves, fails on any homeless-focus frame, and asserts the sweep actually covered the declared control set. Pair it with per-resolution screenshot capture (`get_viewport().get_texture().get_image().save_png(...)`) and a 0.5% pixel-diff gate in CI.
