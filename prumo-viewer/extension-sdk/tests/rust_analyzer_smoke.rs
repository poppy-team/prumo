use prumo_extension_sdk::lsp::LspClient;
use std::fs;
use std::path::Path;
use std::process::Command;
use std::thread;
use std::time::{Duration, Instant};

#[test]
fn real_rust_analyzer_completes_and_reports_notifications() {
    if std::env::var("PRUMO_RUN_RUST_ANALYZER_SMOKE").as_deref() != Ok("1") {
        return;
    }
    assert!(
        Command::new("rust-analyzer")
            .arg("--version")
            .status()
            .is_ok_and(|status| status.success())
    );

    let workspace = tempfile::tempdir().unwrap();
    let source_path = workspace.path().join("src/main.rs");
    fs::create_dir_all(source_path.parent().unwrap()).unwrap();
    fs::write(
        workspace.path().join("Cargo.toml"),
        "[package]\nname = \"lsp-smoke\"\nversion = \"0.1.0\"\nedition = \"2024\"\n",
    )
    .unwrap();
    let source = "struct Demo { field: usize }\nfn main() {\n    let value = Demo { field: 1 };\n    value.\n}\n";
    fs::write(&source_path, source).unwrap();

    let mut client = LspClient::start(
        "rust-analyzer",
        &[],
        workspace.path(),
        Duration::from_secs(10),
    )
    .expect("rust-analyzer should start");
    client
        .did_open(&source_path.to_string_lossy(), "rust", source, 1)
        .unwrap();

    let mut completion = None;
    let completion_started = Instant::now();
    let deadline = Instant::now() + Duration::from_secs(15);
    while Instant::now() < deadline {
        match client.completion(
            &source_path.to_string_lossy(),
            4,
            10,
            Duration::from_secs(2),
        ) {
            Ok(response) if !response.clone().items().is_empty() => {
                completion = Some(response);
                break;
            }
            Ok(_) => thread::sleep(Duration::from_millis(100)),
            Err(error) if Instant::now() < deadline => {
                eprintln!("rust-analyzer completion retry: {error}");
                thread::sleep(Duration::from_millis(100));
            }
            Err(error) => panic!("rust-analyzer completion failed: {error}"),
        }
    }
    eprintln!(
        "rust-analyzer completion latency: {:?}",
        completion_started.elapsed()
    );
    assert!(
        completion
            .is_some_and(|response| response.items().iter().any(|item| item.label == "field")),
        "rust-analyzer did not return the Demo.field completion"
    );

    let mut diagnostics = Vec::new();
    let deadline = Instant::now() + Duration::from_secs(5);
    while Instant::now() < deadline && diagnostics.is_empty() {
        diagnostics.extend(client.drain_notifications());
        if diagnostics.is_empty() {
            thread::sleep(Duration::from_millis(50));
        }
    }
    let has_publish_diagnostics = diagnostics.iter().any(|notification| {
        notification
            .get("method")
            .and_then(|method| method.as_str())
            == Some("textDocument/publishDiagnostics")
    });
    let has_sysroot_notice = diagnostics.iter().any(|notification| {
        notification
            .get("method")
            .and_then(|method| method.as_str())
            == Some("window/showMessage")
            && notification
                .get("params")
                .and_then(|params| params.get("message"))
                .and_then(|message| message.as_str())
                .is_some_and(|message| message.contains("sysroot"))
    });
    eprintln!(
        "rust-analyzer diagnostics notification: publish={has_publish_diagnostics}, sysroot_notice={has_sysroot_notice}"
    );
    assert!(
        has_publish_diagnostics || has_sysroot_notice,
        "rust-analyzer did not publish diagnostics or explain the missing sysroot: {diagnostics:?}"
    );
    assert!(Path::new(&source_path).is_file());
}
