# Native Workspace Viewer (`prumo-viewer`)

The **Prumo Native Workspace Viewer** is a graphical and terminal visualization component implemented in Rust for instant navigation of massive repositories.

## Why Rust?

In projects with tens of thousands of files, hundreds of Goals, and complex dependency graphs, interfaces based on Electron or interpreted scripts suffer from excessive memory consumption and rendering latency. `prumo-viewer` delivers:

- **Sub-Millisecond Startup Time**: Direct filesystem access with a shared in-memory cache.
- **Fast DAG Rendering**: Interactive visualization of task graphs without freezes.
- **Extension Ecosystem**: A secure bus for custom visualization plugins.
- **Terminal Integration**: Works in harmony with `prumo-agent` (pa).

## Invoking the Viewer

```bash
# Start the native viewer in the current repository
prumo-viewer .

# Start focused on the graph of a specific Goal
prumo-viewer . --goal P01-G01
```
