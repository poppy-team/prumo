use freya::prelude::*;
use torin::prelude::Direction;

use crate::{
    WorkspaceCommands,
    state::{
        AgentFileStatus, AppState, ChangedFile, DockView, FileKind, NoticeTone, SidebarView,
        TreeNode,
    },
    ui::icons::icon,
};

#[derive(PartialEq)]
pub struct ActivityBar {
    pub state: State<AppState>,
}

impl Component for ActivityBar {
    fn render(&self) -> impl IntoElement {
        let state = self.state;
        let colors = use_theme().read().colors.clone();
        let count = state.read().changed_files.len();

        rect()
            .width(Size::px(44.))
            .height(Size::fill())
            .background(colors.surface_inverse)
            .border(Border::new().fill(colors.border).width(BorderWidth {
                top: 0.,
                right: 1.,
                bottom: 0.,
                left: 0.,
            }))
            .padding(Gaps::new(0., 6., 0., 6.))
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::fill())
                    .vertical()
                    .padding(Gaps::new(0., 4., 0., 4.))
                    .spacing(4.)
                    .child(ActivityRailItem {
                        state,
                        icon: "files",
                        label: "Explorer",
                        badge: None,
                        active: state.read().sidebar_visible
                            && (!state.read().agent_panel_visible
                                || state.read().viewport_width >= 1180.)
                            && state.read().sidebar_view == SidebarView::Explorer,
                        action: RailAction::Explorer,
                    })
                    .child(ActivityRailItem {
                        state,
                        icon: "list_checks",
                        label: "Changes",
                        badge: (count > 0).then_some(count),
                        active: state.read().sidebar_visible
                            && (!state.read().agent_panel_visible
                                || state.read().viewport_width >= 1180.)
                            && state.read().sidebar_view == SidebarView::Changes,
                        action: RailAction::Changes,
                    })
                    .child(ActivityRailItem {
                        state,
                        icon: "bot",
                        label: "Prumo Agent",
                        badge: None,
                        active: state.read().agent_panel_visible
                            && state.read().viewport_width >= 900.,
                        action: RailAction::Agent,
                    })
                    .child(ActivityRailItem {
                        state,
                        icon: "terminal",
                        label: "Integrated Terminal",
                        badge: None,
                        active: state.read().dock_open
                            && state.read().dock_view == DockView::Terminal,
                        action: RailAction::Terminal,
                    })
                    .child(ActivityRailItem {
                        state,
                        icon: "settings",
                        label: "Settings",
                        badge: None,
                        active: state.read().config_open,
                        action: RailAction::Settings,
                    }),
            )
    }
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum RailAction {
    Explorer,
    Changes,
    Agent,
    Terminal,
    Settings,
}

#[derive(PartialEq)]
struct ActivityRailItem {
    state: State<AppState>,
    icon: &'static str,
    label: &'static str,
    badge: Option<usize>,
    active: bool,
    action: RailAction,
}

impl Component for ActivityRailItem {
    fn render(&self) -> impl IntoElement {
        let a11y_id = use_a11y();
        let focus = use_focus(a11y_id);
        let colors = use_theme().read().colors.clone();
        let mut state = self.state;
        let action = self.action;
        let foreground = if self.active {
            colors.primary
        } else {
            colors.text_secondary
        };
        let background = if self.active {
            colors.surface_tertiary
        } else {
            Color::TRANSPARENT
        };
        let focus_width = if focus() == Focus::Keyboard { 2. } else { 0. };

        rect()
            .width(Size::fill())
            .height(Size::px(36.))
            .a11y_id(a11y_id)
            .a11y_focusable(true)
            .a11y_role(AccessibilityRole::Button)
            .a11y_alt(self.label)
            .background(background)
            .corner_radius(CornerRadius::new_all(5.))
            .border(Border::new().fill(colors.primary).width(focus_width))
            .vertical()
            .main_align(Alignment::center())
            .cross_align(Alignment::center())
            .on_press(move |_| {
                let mut state = state.write();
                match action {
                    RailAction::Explorer => {
                        if state.viewport_width < 1180. {
                            state.sidebar_view = SidebarView::Explorer;
                            state.sidebar_visible = true;
                            state.agent_panel_visible = false;
                        } else if state.sidebar_visible
                            && state.sidebar_view == SidebarView::Explorer
                        {
                            state.sidebar_visible = false;
                        } else {
                            state.sidebar_view = SidebarView::Explorer;
                            state.sidebar_visible = true;
                        }
                    }
                    RailAction::Changes => {
                        if state.viewport_width < 1180. {
                            state.sidebar_view = SidebarView::Changes;
                            state.sidebar_visible = true;
                            state.agent_panel_visible = false;
                        } else if state.sidebar_visible
                            && state.sidebar_view == SidebarView::Changes
                        {
                            state.sidebar_visible = false;
                        } else {
                            state.sidebar_view = SidebarView::Changes;
                            state.sidebar_visible = true;
                        }
                    }
                    RailAction::Agent => {
                        if state.viewport_width < 1180. {
                            state.sidebar_visible = false;
                            state.agent_panel_visible = true;
                        } else {
                            let is_visible = state.agent_panel_visible;
                            state.agent_panel_visible = !is_visible;
                        }
                    }
                    RailAction::Terminal => {
                        state.dock_view = DockView::Terminal;
                        state.dock_open = true;
                    }
                    RailAction::Settings => {
                        state.config_open = true;
                    }
                }
            })
            .child(
                SvgViewer::new((self.icon, icon(self.icon)))
                    .color(foreground)
                    .width(Size::px(19.))
                    .height(Size::px(19.)),
            )
            .maybe(self.badge.is_some(), |rect| {
                rect.child(
                    label()
                        .color(colors.primary)
                        .font_size(8.)
                        .font_weight(FontWeight::BOLD)
                        .text(self.badge.unwrap_or_default().to_string()),
                )
            })
    }
}

#[derive(PartialEq)]
pub struct Sidebar {
    pub state: State<AppState>,
    pub workspace: WorkspaceCommands,
}

impl Component for Sidebar {
    fn render(&self) -> impl IntoElement {
        match self.state.read().sidebar_view {
            SidebarView::Explorer => {
                rect()
                    .width(Size::fill())
                    .height(Size::fill())
                    .child(ExplorerView {
                        state: self.state,
                        workspace: self.workspace,
                    })
            }
            SidebarView::Changes => {
                rect()
                    .width(Size::fill())
                    .height(Size::fill())
                    .child(ChangesView {
                        state: self.state,
                        workspace: self.workspace,
                    })
            }
        }
    }
}

#[derive(PartialEq)]
struct ExplorerView {
    state: State<AppState>,
    workspace: WorkspaceCommands,
}

impl Component for ExplorerView {
    fn render(&self) -> impl IntoElement {
        let state = self.state;
        let root = state.read().tree.clone();
        let workspace = self.workspace;
        let colors = use_theme().read().colors.clone();
        let mut root_children = rect().width(Size::fill()).vertical();
        for child in root.children {
            root_children = root_children.child(TreeView {
                state,
                node: child,
                depth: 0,
                workspace,
            });
        }

        rect()
            .width(Size::fill())
            .height(Size::fill())
            .background(colors.surface_primary)
            .vertical()
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(36.))
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .padding(Gaps::new(10., 0., 6., 0.))
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(11.)
                            .font_weight(FontWeight::BOLD)
                            .text("EXPLORER"),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(24.))
                            .padding(Gaps::new(6., 0., 6., 0.))
                            .on_press(move |_| workspace.refresh())
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.)
                                    .text("REFRESH"),
                            ),
                    ),
            )
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(1.))
                    .background(colors.border),
            )
            .child(
                ScrollView::new().direction(Direction::Vertical).child(
                    rect()
                        .width(Size::fill())
                        .vertical()
                        .padding(Gaps::new(0., 4., 0., 4.))
                        .child(WorkspaceRow { state })
                        .child(root_children),
                ),
            )
    }
}

#[derive(PartialEq)]
struct WorkspaceRow {
    state: State<AppState>,
}

impl Component for WorkspaceRow {
    fn render(&self) -> impl IntoElement {
        let colors = use_theme().read().colors.clone();
        let name = self.state.read().workspace_name.clone();

        rect()
            .width(Size::fill())
            .height(Size::px(26.))
            .horizontal()
            .cross_align(Alignment::center())
            .spacing(6.)
            .padding(Gaps::new(8., 0., 8., 0.))
            .child(
                SvgViewer::new(("folder", icon("folder")))
                    .color(colors.text_secondary)
                    .width(Size::px(14.))
                    .height(Size::px(14.)),
            )
            .child(
                label()
                    .color(colors.text_primary)
                    .font_size(12.)
                    .font_weight(FontWeight::BOLD)
                    .text(name.to_uppercase()),
            )
            .child(
                rect().width(Size::fill()).child(
                    label()
                        .color(colors.text_secondary)
                        .font_size(12.)
                        .text("⌄"),
                ),
            )
    }
}

#[derive(PartialEq)]
struct TreeView {
    state: State<AppState>,
    node: TreeNode,
    depth: usize,
    workspace: WorkspaceCommands,
}

impl Component for TreeView {
    fn render(&self) -> impl IntoElement {
        let children = if self.node.expanded {
            self.node.children.clone()
        } else {
            Vec::new()
        };
        let has_children = !self.node.children.is_empty();
        let mut child_column = rect().width(Size::fill()).vertical();
        for child in children {
            child_column = child_column.child(TreeView {
                state: self.state,
                node: child,
                depth: self.depth + 1,
                workspace: self.workspace,
            });
        }

        rect()
            .width(Size::fill())
            .vertical()
            .child(FileTreeRow {
                state: self.state,
                node: self.node.clone(),
                depth: self.depth,
                workspace: self.workspace,
            })
            .maybe(has_children && self.node.expanded, |rect| {
                rect.child(child_column)
            })
    }
}

#[derive(PartialEq)]
struct FileTreeRow {
    state: State<AppState>,
    node: TreeNode,
    depth: usize,
    workspace: WorkspaceCommands,
}

impl Component for FileTreeRow {
    fn render(&self) -> impl IntoElement {
        let mut state = self.state;
        let node = self.node.clone();
        let path = node.path.clone();
        let has_children = !node.children.is_empty();
        let is_dir = node.is_dir;
        let is_open = state
            .read()
            .active_tab()
            .is_some_and(|tab| tab.path == path);
        let workspace = self.workspace;
        let colors = use_theme().read().colors.clone();
        let status = node.agent_status;
        let depth_padding = 4. + self.depth as f32 * 12.;

        Button::new()
            .flat()
            .expanded()
            .height(Size::px(24.))
            .padding(0.)
            .on_press(move |_| {
                if is_dir {
                    crate::services::workspace::toggle_folder(&mut state.write().tree, &path);
                } else {
                    let already_open = state.write().activate_path(&path);
                    if !already_open {
                        state.write().selected_path = Some(path.clone());
                        workspace.load_file(path.clone());
                    }
                }
            })
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::fill())
                    .horizontal()
                    .cross_align(Alignment::center())
                    .spacing(5.)
                    .padding(Gaps::new(depth_padding, 0., 6., 0.))
                    .child(rect().width(Size::px(12.)).maybe(has_children, |rect| {
                        rect.child(
                            SvgViewer::new((
                                if node.expanded {
                                    "chevron_down"
                                } else {
                                    "chevron_right"
                                },
                                icon(if node.expanded {
                                    "chevron_down"
                                } else {
                                    "chevron_right"
                                }),
                            ))
                            .color(colors.text_secondary)
                            .width(Size::px(12.))
                            .height(Size::px(12.)),
                        )
                    }))
                    .child(
                        SvgViewer::new((file_icon(node.kind), icon(file_icon(node.kind))))
                            .color(file_icon_color(&colors, node.kind))
                            .width(Size::px(14.))
                            .height(Size::px(14.)),
                    )
                    .child(
                        rect().width(Size::fill()).child(
                            label()
                                .color(if is_open {
                                    colors.text_primary
                                } else {
                                    colors.text_secondary
                                })
                                .font_size(12.)
                                .max_lines(1)
                                .text(node.name),
                        ),
                    )
                    .maybe(status.is_some(), |rect| {
                        rect.child(
                            label()
                                .color(agent_status_color(&colors, status.unwrap()))
                                .font_size(10.)
                                .font_weight(FontWeight::BOLD)
                                .text(status.unwrap().marker()),
                        )
                    }),
            )
    }
}

#[derive(PartialEq)]
struct ChangesView {
    state: State<AppState>,
    workspace: WorkspaceCommands,
}

impl Component for ChangesView {
    fn render(&self) -> impl IntoElement {
        let state = self.state;
        let changed_files = state.read().changed_files.clone();
        let colors = use_theme().read().colors.clone();
        let mut rows = Vec::new();
        for file in changed_files.iter().cloned() {
            rows.push(
                ChangeRow {
                    state,
                    file,
                    workspace: self.workspace,
                }
                .into_element(),
            );
        }
        let row_count = rows.len();
        let content = if rows.is_empty() {
            EmptyState {
                icon: "circle_alert",
                title: "No changes",
                detail: "Agent file events appear here.",
            }
            .into_element()
        } else {
            ScrollView::new()
                .direction(Direction::Vertical)
                .children(rows)
                .into_element()
        };

        rect()
            .width(Size::fill())
            .height(Size::fill())
            .background(colors.surface_primary)
            .vertical()
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(36.))
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .padding(Gaps::new(12., 0., 12., 0.))
                    .child(
                        rect()
                            .horizontal()
                            .cross_align(Alignment::center())
                            .spacing(7.)
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(11.)
                                    .font_weight(FontWeight::BOLD)
                                    .text("CHANGES"),
                            )
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.)
                                    .text(row_count.to_string()),
                            ),
                    )
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(10.)
                            .text("CURRENT RUN"),
                    ),
            )
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(1.))
                    .background(colors.border),
            )
            .child(content)
    }
}

#[derive(PartialEq)]
struct ChangeRow {
    state: State<AppState>,
    file: ChangedFile,
    workspace: WorkspaceCommands,
}

impl Component for ChangeRow {
    fn render(&self) -> impl IntoElement {
        let mut state = self.state;
        let file = self.file.clone();
        let status = file.status;
        let path = std::path::PathBuf::from(&file.path);
        let deleted_path = file.path.clone();
        let workspace = self.workspace;
        let colors = use_theme().read().colors.clone();
        let foreground = match status {
            AgentFileStatus::Created => colors.success,
            AgentFileStatus::Modified => colors.warning,
            AgentFileStatus::Deleted => colors.error,
        };

        Button::new()
            .flat()
            .expanded()
            .height(Size::px(28.))
            .padding(0.)
            .on_press(move |_| {
                if status == AgentFileStatus::Deleted {
                    state.write().show_notice(
                        NoticeTone::Error,
                        format!("{} was deleted by the active run", deleted_path),
                    );
                } else {
                    let absolute_path = state.read().workspace_root.join(&path);
                    workspace.load_file(absolute_path);
                }
            })
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::fill())
                    .horizontal()
                    .cross_align(Alignment::center())
                    .spacing(8.)
                    .padding(Gaps::new(10., 0., 10., 0.))
                    .child(
                        label()
                            .color(foreground)
                            .font_size(11.)
                            .font_weight(FontWeight::BOLD)
                            .text(status.marker()),
                    )
                    .child(
                        rect().width(Size::fill()).child(
                            label()
                                .color(colors.text_primary)
                                .font_size(12.)
                                .max_lines(1)
                                .text(file.path),
                        ),
                    ),
            )
    }
}

#[derive(PartialEq)]
struct EmptyState {
    icon: &'static str,
    title: &'static str,
    detail: &'static str,
}

impl Component for EmptyState {
    fn render(&self) -> impl IntoElement {
        let colors = use_theme().read().colors.clone();

        rect()
            .width(Size::fill())
            .vertical()
            .main_align(Alignment::center())
            .padding(Gaps::new(18., 34., 18., 34.))
            .spacing(8.)
            .child(
                SvgViewer::new((self.icon, icon(self.icon)))
                    .color(colors.text_secondary)
                    .width(Size::px(24.))
                    .height(Size::px(24.)),
            )
            .child(
                label()
                    .color(colors.text_primary)
                    .font_size(12.)
                    .font_weight(FontWeight::BOLD)
                    .text(self.title),
            )
            .child(
                label()
                    .color(colors.text_secondary)
                    .font_size(11.)
                    .text_align(TextAlign::Center)
                    .text(self.detail),
            )
    }
}

fn file_icon(kind: FileKind) -> &'static str {
    match kind {
        FileKind::Folder => "folder",
        FileKind::Rust | FileKind::Go | FileKind::TypeScript | FileKind::JavaScript => "file_code",
        FileKind::Markdown => "file_text",
        FileKind::Json | FileKind::Yaml | FileKind::Toml | FileKind::Manifest => "package",
        FileKind::Other => "file_text",
    }
}

fn file_icon_color(colors: &ColorsSheet, kind: FileKind) -> Color {
    match kind {
        FileKind::Go => colors.secondary,
        FileKind::Rust => colors.warning,
        FileKind::Manifest | FileKind::Toml => colors.primary,
        FileKind::Markdown | FileKind::Other => colors.text_secondary,
        FileKind::Json | FileKind::Yaml => colors.tertiary,
        FileKind::TypeScript => colors.primary,
        FileKind::JavaScript => colors.warning,
        FileKind::Folder => colors.text_secondary,
    }
}

fn agent_status_color(colors: &ColorsSheet, status: AgentFileStatus) -> Color {
    match status {
        AgentFileStatus::Created => colors.success,
        AgentFileStatus::Modified => colors.warning,
        AgentFileStatus::Deleted => colors.error,
    }
}
