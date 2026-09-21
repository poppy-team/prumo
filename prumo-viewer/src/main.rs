use prumo_viewer::app;
use prumo_viewer::config::{default_socket_path, resolve_renderer, CliArgs, GraphicsConfig};

fn main() -> eframe::Result {
    let args = CliArgs::parse();
    let config = GraphicsConfig::default();

    let socket_path = args.socket_path.clone().unwrap_or_else(default_socket_path);
    let workspace_root = args.workspace_dir.clone();

    // P0.6: Shared Tokio runtime with bounded workers (2 threads)
    let rt = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(2)
        .enable_all()
        .build()
        .expect("Failed to initialize shared Tokio runtime");
    let _guard = rt.enter();

    // Select renderer according to policy: Glow (OpenGL baseline) is default
    let renderer = resolve_renderer(&config, &args);
    let renderer_name = match renderer {
        eframe::Renderer::Glow => "Glow (OpenGL)",
        eframe::Renderer::Wgpu => "WGPU",
    }
    .to_string();

    let native_options = eframe::NativeOptions {
        viewport: egui::ViewportBuilder::default()
            .with_inner_size([1360.0, 840.0])
            .with_min_inner_size([900.0, 600.0])
            .with_title("Prumo Native Workspace Viewer"),
        renderer,
        ..Default::default()
    };

    let app_workspace = workspace_root.clone();
    let app_socket = socket_path.clone();
    let app_renderer = renderer_name.clone();

    eframe::run_native(
        "Prumo Native",
        native_options,
        Box::new(move |_cc| {
            Ok(Box::new(app::WorkspaceViewerApp::new(
                app_workspace,
                app_socket,
                app_renderer,
            )))
        }),
    )
}
