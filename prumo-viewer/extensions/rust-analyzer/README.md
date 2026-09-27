# Rust Analyzer

Reference LSP package for the Prumo Viewer Extension Contract v1.

The viewer starts `rust-analyzer` only after the workspace grants `editor.lsp` to this package. The package does not bundle a language server or auto-approve its high-risk capability.

Install `rust-analyzer` and make it available on `PATH`. Keep `rustc` and `rust-analyzer` on the same Rust toolchain. Install the `rust-src` Rust component when full standard-library diagnostics are required.
