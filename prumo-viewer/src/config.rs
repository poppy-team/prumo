use std::path::PathBuf;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "lowercase")]
pub enum GraphicsRenderer {
    #[default]
    Glow,
    Wgpu,
    Auto,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "lowercase")]
pub enum WgpuBackendPreference {
    #[default]
    Auto,
    Gl,
    Vulkan,
    Dx12,
    Metal,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GraphicsConfig {
    pub renderer: GraphicsRenderer,
    pub wgpu_backend: WgpuBackendPreference,
    pub fallback: bool,
    pub last_boot_ok: bool,
    pub last_backend: String,
}

impl Default for GraphicsConfig {
    fn default() -> Self {
        Self {
            renderer: GraphicsRenderer::Glow,
            wgpu_backend: WgpuBackendPreference::Auto,
            fallback: true,
            last_boot_ok: true,
            last_backend: "opengl".to_string(),
        }
    }
}

#[derive(Debug, Clone)]
pub struct CliArgs {
    pub renderer_override: Option<GraphicsRenderer>,
    pub wgpu_backend_override: Option<WgpuBackendPreference>,
    pub safe_graphics: bool,
    pub socket_path: Option<PathBuf>,
    pub workspace_dir: PathBuf,
}

impl CliArgs {
    pub fn parse() -> Self {
        let args: Vec<String> = std::env::args().collect();
        let mut renderer_override = None;
        let mut wgpu_backend_override = None;
        let mut safe_graphics = false;
        let mut socket_path = None;
        let mut workspace_dir = std::env::current_dir().unwrap_or_else(|_| PathBuf::from("."));

        let mut i = 1;
        while i < args.len() {
            match args[i].as_str() {
                "--help" | "-h" => {
                    println!("Usage: prumo-native [OPTIONS] [WORKSPACE_DIR]\n");
                    println!("Prumo Native Workspace Viewer: Agent-Aware Lightweight Workspace Viewer & Editor\n");
                    println!("Options:");
                    println!("  --renderer <glow|wgpu|auto>     Graphics rendering backend (default: glow)");
                    println!("  --wgpu-backend <gl|vulkan|dx12|metal|auto>  WGPU backend preference");
                    println!("  --safe-graphics                 Enforce Glow/OpenGL fallback mode");
                    println!("  --socket <PATH>                 Path to Prumo daemon Unix socket");
                    println!("  --workspace <DIR>               Workspace directory to view");
                    println!("  -h, --help                      Print this help message");
                    std::process::exit(0);
                }
                "--renderer" if i + 1 < args.len() => {
                    i += 1;
                    match args[i].to_lowercase().as_str() {
                        "glow" | "opengl" => renderer_override = Some(GraphicsRenderer::Glow),
                        "wgpu" => renderer_override = Some(GraphicsRenderer::Wgpu),
                        "auto" => renderer_override = Some(GraphicsRenderer::Auto),
                        _ => eprintln!("Warning: unknown renderer '{}', using auto", args[i]),
                    }
                }
                "--wgpu-backend" if i + 1 < args.len() => {
                    i += 1;
                    match args[i].to_lowercase().as_str() {
                        "gl" => wgpu_backend_override = Some(WgpuBackendPreference::Gl),
                        "vulkan" => wgpu_backend_override = Some(WgpuBackendPreference::Vulkan),
                        "dx12" => wgpu_backend_override = Some(WgpuBackendPreference::Dx12),
                        "metal" => wgpu_backend_override = Some(WgpuBackendPreference::Metal),
                        "auto" => wgpu_backend_override = Some(WgpuBackendPreference::Auto),
                        _ => eprintln!("Warning: unknown wgpu backend '{}'", args[i]),
                    }
                }
                "--safe-graphics" => {
                    safe_graphics = true;
                }
                "--socket" if i + 1 < args.len() => {
                    i += 1;
                    socket_path = Some(PathBuf::from(&args[i]));
                }
                "--workspace" if i + 1 < args.len() => {
                    i += 1;
                    workspace_dir = PathBuf::from(&args[i]);
                }
                arg if !arg.starts_with('-') && i == 1 => {
                    workspace_dir = PathBuf::from(arg);
                }
                _ => {}
            }
            i += 1;
        }

        Self {
            renderer_override,
            wgpu_backend_override,
            safe_graphics,
            socket_path,
            workspace_dir,
        }
    }
}

pub fn default_socket_path() -> PathBuf {
    // 1. Local workspace .prumo/prumo.sock
    let local = PathBuf::from(".prumo/prumo.sock");
    if local.exists() {
        return local;
    }

    // 2. User home ~/.prumo/prumo.sock
    if let Some(home) = dirs::home_dir() {
        let home_sock = home.join(".prumo").join("prumo.sock");
        if home_sock.exists() {
            return home_sock;
        }
    }

    // Default fallback
    local
}

pub fn resolve_renderer(config: &GraphicsConfig, cli: &CliArgs) -> eframe::Renderer {
    if cli.safe_graphics {
        eprintln!("[prumo-native] --safe-graphics active: enforcing Glow/OpenGL renderer");
        return eframe::Renderer::Glow;
    }

    let target = cli.renderer_override.unwrap_or(config.renderer);
    match target {
        GraphicsRenderer::Glow => {
            eprintln!("[prumo-native] Selected renderer: Glow (OpenGL baseline, lightweight)");
            eframe::Renderer::Glow
        }
        GraphicsRenderer::Wgpu => {
            eprintln!("[prumo-native] Selected renderer: WGPU (modern GPU pipeline)");
            eframe::Renderer::Wgpu
        }
        GraphicsRenderer::Auto => {
            // Default baseline according to Constituição 82: Glow/OpenGL
            eprintln!("[prumo-native] Renderer Auto: defaulting to Glow (OpenGL)");
            eframe::Renderer::Glow
        }
    }
}
