use std::path::{Path, PathBuf};
use std::time::Instant;
use egui::{Color32, FontId, RichText, ScrollArea, Stroke, Ui};
use tokio::sync::mpsc;

use crate::client::{Event, PrumoClient, RunStatus, StartRequest};
use crate::diff::{DiffLineTag, DiffService, FileDiff, MergeResult};
use crate::document::DocumentStore;
use crate::follow::AgentFollowController;
use crate::projection::RunFileProjection;
use crate::terminal::LazyTerminal;
use crate::watcher::{WorkspaceEvent, WorkspaceWatcher};
use crate::workspace::{CachedWorkspaceTree, FileNode};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CenterViewMode {
    Code,
    Diff,
    Conflict,
}

pub struct WorkspaceViewerApp {
    pub workspace_root: PathBuf,
    pub client: PrumoClient,
    pub is_connected: bool,
    pub active_run_id: Option<String>,
    pub known_runs: Vec<RunStatus>,

    // Projections & Services
    pub tree: CachedWorkspaceTree,
    pub watcher: Option<WorkspaceWatcher>,
    pub doc_store: DocumentStore,
    pub projection: RunFileProjection,
    pub follow: AgentFollowController,
    pub terminal: LazyTerminal,

    // Event streaming
    pub event_rx: Option<mpsc::UnboundedReceiver<Event>>,
    pub timeline_events: Vec<Event>,

    // UI state
    pub view_mode: CenterViewMode,
    pub show_terminal: bool,
    pub show_sidebar: bool,
    pub show_timeline: bool,
    pub active_diff: Option<FileDiff>,
    pub active_conflict: Option<String>,
    pub pending_permission_req: Option<String>,
    pub status_message: String,
    pub renderer_name: String,
    pub last_status_poll: Instant,
}

impl WorkspaceViewerApp {
    pub fn new(
        workspace_root: PathBuf,
        socket_path: PathBuf,
        renderer_name: String,
    ) -> Self {
        let client = PrumoClient::new(&socket_path);
        let is_connected = client.is_alive();
        let tree = CachedWorkspaceTree::new(&workspace_root);
        let watcher = WorkspaceWatcher::new(&workspace_root).ok();

        let mut app = Self {
            workspace_root: workspace_root.clone(),
            client,
            is_connected,
            active_run_id: None,
            known_runs: Vec::new(),
            tree,
            watcher,
            doc_store: DocumentStore::new(),
            projection: RunFileProjection::new(""),
            follow: AgentFollowController::new(),
            terminal: LazyTerminal::new(),
            event_rx: None,
            timeline_events: Vec::new(),
            view_mode: CenterViewMode::Code,
            show_terminal: false,
            show_sidebar: true,
            show_timeline: true,
            active_diff: None,
            active_conflict: None,
            pending_permission_req: None,
            status_message: "Ready".to_string(),
            renderer_name,
            last_status_poll: Instant::now(),
        };

        // If connected, fetch initial runs
        if app.is_connected {
            app.refresh_runs();
        }

        app
    }

    pub fn refresh_runs(&mut self) {
        if let Ok(runs) = self.client.list() {
            self.known_runs = runs.clone();
            if self.active_run_id.is_none() {
                if let Some(active) = runs.iter().find(|r| r.active) {
                    self.select_run(&active.run_id);
                } else if let Some(first) = runs.first() {
                    self.select_run(&first.run_id);
                }
            }
        }
    }

    pub fn select_run(&mut self, run_id: &str) {
        self.active_run_id = Some(run_id.to_string());
        self.projection.reset(run_id);
        self.timeline_events.clear();

        // Replay existing events
        if let Ok(evs) = self.client.events(run_id) {
            for ev in &evs {
                self.projection.process_event(ev, &self.workspace_root);
            }
            self.timeline_events = evs;
        }

        // Open push subscription stream from daemon
        let from = self.timeline_events.len();
        if let Ok(rx) = self.client.subscribe(run_id.to_string(), from) {
            self.event_rx = Some(rx);
        }
    }

    pub fn start_new_run(&mut self, goal: &str) {
        let req = StartRequest {
            goal: goal.to_string(),
            workspace: self.workspace_root.to_string_lossy().to_string(),
            ..Default::default()
        };

        match self.client.start(req) {
            Ok(run_id) => {
                self.status_message = format!("Started run {}", run_id);
                self.refresh_runs();
                self.select_run(&run_id);
            }
            Err(e) => {
                self.status_message = format!("Start run failed: {}", e);
            }
        }
    }

    pub fn cancel_active_run(&mut self) {
        if let Some(run_id) = &self.active_run_id {
            let _ = self.client.cancel(run_id);
            self.status_message = format!("Cancelled run {}", run_id);
            self.refresh_runs();
        }
    }

    pub fn open_diff_for_active(&mut self) {
        if let Some(doc) = self.doc_store.active_document() {
            let diff = FileDiff::compute(&doc.saved_content, &doc.content);
            self.active_diff = Some(diff);
            self.view_mode = CenterViewMode::Diff;
        }
    }

    pub fn poll_background_events(&mut self, ctx: &egui::Context) {
        let mut had_activity = false;

        // 1. Filesystem events from watcher
        if let Some(w) = &self.watcher {
            let file_events = w.poll_events();
            for ev in file_events {
                had_activity = true;
                self.tree.apply_event(&ev);

                // External change check for open documents
                match ev {
                    WorkspaceEvent::Modified(path) => {
                        if let Some(doc) = self.doc_store.get_document(&path) {
                            if doc.is_dirty {
                                // Conflict! Human has unsaved edits and disk changed!
                                if let Ok(disk_content) = std::fs::read_to_string(&path) {
                                    let merge = DiffService::three_way_merge(
                                        &doc.saved_content,
                                        &doc.content,
                                        &disk_content,
                                    );
                                    if let MergeResult::Conflict { merged_with_markers, .. } = merge {
                                        self.active_conflict = Some(merged_with_markers);
                                        self.view_mode = CenterViewMode::Conflict;
                                    }
                                }
                            } else {
                                let _ = self.doc_store.reload_if_clean(&path);
                            }
                        }
                    }
                    _ => {}
                }
            }
        }

        // 2. Incoming push stream events from Prumo daemon
        if let Some(rx) = &mut self.event_rx {
            while let Ok(ev) = rx.try_recv() {
                had_activity = true;
                self.projection.process_event(&ev, &self.workspace_root);
                self.follow.handle_event(&ev, &self.workspace_root, &mut self.doc_store);

                if ev.kind == "permission.requested" {
                    if let Some(req_id) = ev.payload.get("request_id").and_then(|v| v.as_str()) {
                        self.pending_permission_req = Some(req_id.to_string());
                    }
                }

                self.timeline_events.push(ev);
            }
        }

        // 3. Periodic connection check (every 3 seconds)
        if self.last_status_poll.elapsed() > std::time::Duration::from_secs(3) {
            self.last_status_poll = Instant::now();
            let alive = self.client.is_alive();
            if alive != self.is_connected {
                self.is_connected = alive;
                had_activity = true;
                if self.is_connected {
                    self.refresh_runs();
                }
            }
        }

        // Event-driven idle: ONLY request repaint if there was real background activity
        if had_activity {
            ctx.request_repaint();
        }
    }
}

impl eframe::App for WorkspaceViewerApp {
    fn logic(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        // Poll background events without continuous painting
        self.poll_background_events(ctx);
    }

    fn ui(&mut self, ui: &mut egui::Ui, _frame: &mut eframe::Frame) {
        // Top Navigation Bar
        egui::Panel::top("header_bar").show(ui, |ui| {
            ui.horizontal(|ui| {
                ui.heading(RichText::new("PRUMO NATIVE").strong().color(Color32::from_rgb(100, 180, 255)));
                ui.separator();

                // Workspace path
                let dir_name = self.workspace_root.file_name().and_then(|s| s.to_str()).unwrap_or(".");
                ui.label(RichText::new(format!("📁 {}", dir_name)).monospace().strong());

                ui.separator();

                // Connection status pill
                if self.is_connected {
                    ui.label(RichText::new("● Daemon Connected").color(Color32::from_rgb(80, 220, 100)));
                } else {
                    ui.label(RichText::new("○ Daemon Offline").color(Color32::from_rgb(220, 80, 80)));
                }

                ui.separator();

                // Renderer badge
                ui.label(RichText::new(format!("⚡ {}", self.renderer_name)).weak());

                ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
                    // Follow Agent Toggle
                    let follow_color = if self.follow.enabled {
                        Color32::from_rgb(80, 220, 150)
                    } else {
                        Color32::GRAY
                    };
                    if ui.button(RichText::new(format!("Follow Agent: {}", if self.follow.enabled { "ON" } else { "OFF" })).color(follow_color)).clicked() {
                        self.follow.toggle();
                    }

                    ui.separator();

                    // View mode buttons
                    if ui.selectable_label(self.view_mode == CenterViewMode::Code, "Code").clicked() {
                        self.view_mode = CenterViewMode::Code;
                    }
                    if ui.selectable_label(self.view_mode == CenterViewMode::Diff, "Diff").clicked() {
                        self.open_diff_for_active();
                    }
                    if self.active_conflict.is_some() {
                        if ui.selectable_label(self.view_mode == CenterViewMode::Conflict, "⚠️ Conflict").clicked() {
                            self.view_mode = CenterViewMode::Conflict;
                        }
                    }

                    ui.separator();

                    // Active Run selector
                    if let Some(run_id) = &self.active_run_id {
                        ui.label(RichText::new(format!("Run: {}", &run_id[..run_id.len().min(12)])).strong().color(Color32::GOLD));
                        if ui.button("Cancel").clicked() {
                            self.cancel_active_run();
                        }
                    } else if ui.button("+ New Run").clicked() {
                        self.start_new_run("Solve pending tasks");
                    }
                });
            });
        });

        // Bottom Status / Terminal Bar
        egui::Panel::bottom("bottom_bar").show(ui, |ui| {
            ui.horizontal(|ui| {
                ui.label(RichText::new(&self.status_message).weak());

                ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
                    if ui.button(if self.show_terminal { "Hide Terminal" } else { "Terminal" }).clicked() {
                        self.show_terminal = !self.show_terminal;
                        if self.show_terminal {
                            let _ = self.terminal.ensure_spawned();
                        }
                    }
                });
            });

            if self.show_terminal {
                ui.separator();
                ScrollArea::vertical().max_height(180.0).show(ui, |ui| {
                    let text = self.terminal.screen_text();
                    ui.label(RichText::new(text).monospace().color(Color32::from_rgb(200, 230, 200)));
                });
            }
        });

        // Left Panel: Explorer & Recent Changes
        if self.show_sidebar {
            egui::Panel::left("explorer_panel").resizable(true).show(ui, |ui| {
                ui.horizontal(|ui| {
                    ui.strong("WORKSPACE EXPLORER");
                    ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
                        if ui.button("⟳").clicked() {
                            self.tree.refresh();
                        }
                    });
                });
                ui.separator();

                ScrollArea::vertical().show(ui, |ui| {
                    render_file_tree_node(ui, &mut self.tree.root_node, &mut self.doc_store, &self.projection);

                    if !self.projection.recent_files.is_empty() {
                        ui.add_space(10.0);
                        ui.separator();
                        ui.strong("AGENT RECENT FILES");
                        for p in &self.projection.recent_files {
                            let fname = p.file_name().and_then(|s| s.to_str()).unwrap_or("?");
                            ui.horizontal(|ui| {
                                if let Some(badge) = self.projection.get_badge(p) {
                                    ui.label(RichText::new(badge.label()).color(badge.color()).strong());
                                }
                                if ui.link(fname).clicked() {
                                    let _ = self.doc_store.open_file(p);
                                }
                            });
                        }
                    }
                });
            });
        }

        // Right Panel: Agent Activity Timeline
        if self.show_timeline {
            egui::Panel::right("agent_panel").resizable(true).show(ui, |ui| {
                ui.horizontal(|ui| {
                    ui.strong("AGENT ACTIVITY");
                    if let Some(working) = &self.projection.currently_working_file {
                        let fname = working.file_name().and_then(|s| s.to_str()).unwrap_or("?");
                        ui.label(RichText::new(format!("● {}", fname)).color(Color32::from_rgb(80, 220, 255)));
                    }
                });
                ui.separator();

                // Permission Approval Banner if waiting
                if let Some(req_id) = self.pending_permission_req.clone() {
                    ui.group(|ui| {
                        ui.label(RichText::new("⚠️ Permission Requested").strong().color(Color32::GOLD));
                        ui.label(format!("Request ID: {}", &req_id[..req_id.len().min(16)]));
                        ui.horizontal(|ui| {
                            if ui.button("✓ Approve").clicked() {
                                let _ = self.client.approve(&req_id);
                                self.pending_permission_req = None;
                            }
                            if ui.button("✕ Deny").clicked() {
                                let _ = self.client.deny(&req_id, "Rejected by user in viewer");
                                self.pending_permission_req = None;
                            }
                        });
                    });
                    ui.separator();
                }

                // Timeline event stream
                ScrollArea::vertical().stick_to_bottom(true).show(ui, |ui| {
                    for ev in &self.timeline_events {
                        render_timeline_event(ui, ev, &mut self.doc_store, &self.workspace_root);
                    }
                });
            });
        }

        // Center Panel: Editor Tabs & Body
        egui::CentralPanel::default().show(ui, |ui| {
            // Tab bar
            ui.horizontal(|ui| {
                let open_tabs = self.doc_store.open_tabs().to_vec();
                for tab_path in open_tabs {
                    let is_active = self.doc_store.active_tab() == Some(&tab_path);
                    let is_dirty = self.doc_store.get_document(&tab_path).map(|d| d.is_dirty).unwrap_or(false);
                    let file_name = tab_path.file_name().and_then(|s| s.to_str()).unwrap_or("?");

                    let title = if is_dirty {
                        format!("● {}", file_name)
                    } else {
                        file_name.to_string()
                    };

                    let text = if is_active {
                        RichText::new(title).strong().color(Color32::WHITE)
                    } else {
                        RichText::new(title).weak()
                    };

                    if ui.selectable_label(is_active, text).clicked() {
                        self.doc_store.set_active(&tab_path);
                        self.view_mode = CenterViewMode::Code;
                    }

                    if ui.small_button("×").clicked() {
                        self.doc_store.close_file(&tab_path);
                    }
                }

                if let Some(doc) = self.doc_store.active_document() {
                    if doc.is_dirty {
                        ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
                            if ui.button("Save (Ctrl+S)").clicked() {
                                let _ = self.doc_store.save_active();
                            }
                        });
                    }
                }
            });

            ui.separator();

            match self.view_mode {
                CenterViewMode::Code => {
                    render_editor_body(ui, &mut self.doc_store, &mut self.follow);
                }
                CenterViewMode::Diff => {
                    render_diff_body(ui, &self.active_diff);
                }
                CenterViewMode::Conflict => {
                    render_conflict_body(ui, &self.active_conflict, &mut self.doc_store);
                }
            }
        });
    }
}

fn render_file_tree_node(
    ui: &mut Ui,
    node: &mut FileNode,
    doc_store: &mut DocumentStore,
    projection: &RunFileProjection,
) {
    if node.is_dir {
        let label = if node.is_expanded {
            format!("▼ {}", node.name)
        } else {
            format!("▶ {}", node.name)
        };

        if ui.selectable_label(false, RichText::new(label).strong()).clicked() {
            node.toggle_expand();
        }

        if node.is_expanded {
            ui.indent(&node.name, |ui| {
                for child in &mut node.children {
                    render_file_tree_node(ui, child, doc_store, projection);
                }
            });
        }
    } else {
        ui.horizontal(|ui| {
            // Live agent badge if present
            if let Some(badge) = projection.get_badge(&node.path) {
                ui.label(RichText::new(badge.label()).color(badge.color()).strong())
                    .on_hover_text(badge.tooltip());
            }

            let is_open = doc_store.active_tab() == Some(&node.path);
            let text = if is_open {
                RichText::new(&node.name).color(Color32::WHITE).strong()
            } else {
                RichText::new(&node.name)
            };

            if ui.link(text).clicked() {
                let _ = doc_store.open_file(&node.path);
            }
        });
    }
}

fn render_editor_body(ui: &mut Ui, store: &mut DocumentStore, follow: &mut AgentFollowController) {
    if let Some(doc) = store.active_document_mut() {
        ScrollArea::both().show(ui, |ui| {
            ui.horizontal(|ui| {
                // Line numbers gutter
                let total_lines = doc.lines_count();
                let gutter_text: String = (1..=total_lines)
                    .map(|n| format!("{:>4}\n", n))
                    .collect();
                ui.label(RichText::new(gutter_text).monospace().weak().color(Color32::from_rgb(120, 130, 140)));

                ui.separator();

                // Editable text area
                let mut text = doc.content.clone();
                let editor_res = ui.add(
                    egui::TextEdit::multiline(&mut text)
                        .font(FontId::monospace(13.0))
                        .code_editor()
                        .desired_width(f32::INFINITY),
                );

                if editor_res.changed() {
                    follow.notify_user_edit_started();
                    doc.set_content(text);
                } else if !editor_res.has_focus() {
                    follow.notify_user_edit_idle();
                }

                // If range is highlighted by agent follow
                if let Some((start, end)) = doc.highlight_range {
                    ui.painter().rect_stroke(
                        editor_res.rect,
                        0.0,
                        Stroke::new(1.0, Color32::from_rgb(80, 200, 255)),
                        egui::StrokeKind::Outside,
                    );
                    ui.label(RichText::new(format!("Agent edited lines {}-{}", start, end)).small().color(Color32::from_rgb(80, 200, 255)));
                }
            });
        });
    } else {
        ui.centered_and_justified(|ui| {
            ui.label(RichText::new("Select a file from the workspace explorer to view and edit.").weak());
        });
    }
}

fn render_diff_body(ui: &mut Ui, diff_opt: &Option<FileDiff>) {
    if let Some(diff) = diff_opt {
        ScrollArea::vertical().show(ui, |ui| {
            ui.horizontal(|ui| {
                ui.label(RichText::new(format!("+{} additions", diff.additions)).color(Color32::from_rgb(80, 220, 100)));
                ui.label(RichText::new(format!("-{} deletions", diff.deletions)).color(Color32::from_rgb(220, 80, 80)));
            });
            ui.separator();

            for line in &diff.lines {
                let (prefix, color) = match line.tag {
                    DiffLineTag::Equal => (" ", Color32::GRAY),
                    DiffLineTag::Insert => ("+", Color32::from_rgb(100, 230, 120)),
                    DiffLineTag::Delete => ("-", Color32::from_rgb(240, 90, 90)),
                };

                let line_str = format!("{} {}", prefix, line.text);
                ui.label(RichText::new(line_str).monospace().color(color));
            }
        });
    } else {
        ui.centered_and_justified(|ui| {
            ui.label("No diff available for current file.");
        });
    }
}

fn render_conflict_body(ui: &mut Ui, conflict_opt: &Option<String>, store: &mut DocumentStore) {
    if let Some(conflict_text) = conflict_opt {
        ui.group(|ui| {
            ui.label(RichText::new("⚠️ CONCURRENT MODIFICATION DETECTED").strong().color(Color32::GOLD));
            ui.label("Both you and the agent edited this file. Please resolve conflict:");
            ui.horizontal(|ui| {
                if ui.button("Accept Agent Version").clicked() {
                    // Accept agent version
                }
                if ui.button("Keep My Edits").clicked() {
                    let _ = store.save_active();
                }
            });
        });
        ui.separator();

        ScrollArea::vertical().show(ui, |ui| {
            ui.label(RichText::new(conflict_text).monospace().color(Color32::GOLD));
        });
    }
}

fn render_timeline_event(ui: &mut Ui, ev: &Event, store: &mut DocumentStore, root: &Path) {
    ui.group(|ui| {
        ui.horizontal(|ui| {
            let (badge, color) = match ev.kind.as_str() {
                "tool_call_delta" | "tool_call_ready" => ("TOOL", Color32::from_rgb(220, 180, 60)),
                "tool_result" => ("OK", Color32::from_rgb(80, 200, 120)),
                "text_delta" => ("CHAT", Color32::from_rgb(100, 180, 255)),
                "run.finished" => ("DONE", Color32::from_rgb(120, 220, 100)),
                "run.failed" => ("FAIL", Color32::from_rgb(255, 70, 70)),
                _ => ("EVT", Color32::GRAY),
            };

            ui.label(RichText::new(badge).strong().color(color));
            ui.label(RichText::new(&ev.kind).small().weak());
        });

        // Display tool path if available
        if let Some(target) = ev.payload.get("path").or_else(|| ev.payload.get("TargetFile")).and_then(|v| v.as_str()) {
            let p = Path::new(target);
            let full_p = if p.is_absolute() { p.to_path_buf() } else { root.join(p) };
            let fname = full_p.file_name().and_then(|s| s.to_str()).unwrap_or(target);

            if ui.link(RichText::new(format!("📄 {}", fname)).small()).clicked() {
                let _ = store.open_file(&full_p);
            }
        }
    });
}
