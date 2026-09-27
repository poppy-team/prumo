use freya::prelude::*;
use torin::prelude::Direction;

use crate::{
    WorkspaceCommands,
    state::{
        AgentFileStatus, AppState, ChangedFile, DockView, ExplorerMenu, FileKind, NoticeTone,
        SidebarView, TreeNode,
    },
    ui::chrome::IconButton,
    ui::git_view::GitView,
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
            .width(Size::px(36.))
            .height(Size::fill())
            .background(colors.background)
            .border(Border::new().fill(colors.border).width(BorderWidth {
                top: 0.,
                right: 1.,
                bottom: 0.,
                left: 0.,
            }))
            .padding(Gaps::new(0., 4., 0., 4.))
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::fill())
                    .vertical()
                    .padding(Gaps::new(0., 2., 0., 2.))
                    .spacing(2.)
                    .child(ActivityRailItem {
                        state,
                        icon: "files",
                        label: "Explorer",
                        badge: false,
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
                        badge: count > 0,
                        active: state.read().sidebar_visible
                            && (!state.read().agent_panel_visible
                                || state.read().viewport_width >= 1180.)
                            && state.read().sidebar_view == SidebarView::Changes,
                        action: RailAction::Changes,
                    })
                    .child(ActivityRailItem {
                        state,
                        icon: "git_branch",
                        label: "Git",
                        badge: !state.read().git.files.is_empty(),
                        active: state.read().sidebar_visible
                            && (!state.read().agent_panel_visible
                                || state.read().viewport_width >= 1180.)
                            && state.read().sidebar_view == SidebarView::Git,
                        action: RailAction::Git,
                    })
                    .child(ActivityRailItem {
                        state,
                        icon: "bot",
                        label: "Prumo Agent",
                        badge: false,
                        active: state.read().agent_panel_visible
                            && state.read().viewport_width >= 900.,
                        action: RailAction::Agent,
                    })
                    .child(ActivityRailItem {
                        state,
                        icon: "terminal",
                        label: "Integrated Terminal",
                        badge: false,
                        active: state.read().dock_open
                            && state.read().dock_view == DockView::Terminal,
                        action: RailAction::Terminal,
                    }),
            )
    }
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum RailAction {
    Explorer,
    Changes,
    Git,
    Agent,
    Terminal,
}

#[derive(PartialEq)]
struct ActivityRailItem {
    state: State<AppState>,
    icon: &'static str,
    label: &'static str,
    badge: bool,
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
        let focus_width = if focus() == Focus::Keyboard { 1. } else { 0. };

        rect()
            .width(Size::fill())
            .height(Size::px(32.))
            .a11y_id(a11y_id)
            .a11y_focusable(true)
            .a11y_role(AccessibilityRole::Button)
            .a11y_alt(self.label)
            .background(Color::TRANSPARENT)
            .corner_radius(CornerRadius::new_all(4.))
            .border(Border::new().fill(colors.border_focus).width(focus_width))
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
                    RailAction::Git => {
                        if state.viewport_width < 1180. {
                            state.sidebar_view = SidebarView::Git;
                            state.sidebar_visible = true;
                            state.agent_panel_visible = false;
                        } else if state.sidebar_visible && state.sidebar_view == SidebarView::Git {
                            state.sidebar_visible = false;
                        } else {
                            state.sidebar_view = SidebarView::Git;
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
                }
            })
            .child(
                SvgViewer::new((self.icon, icon(self.icon)))
                    .color(foreground)
                    .width(Size::px(15.))
                    .height(Size::px(15.)),
            )
            .maybe(self.badge, |el| {
                el.child(
                    rect()
                        .width(Size::px(4.))
                        .height(Size::px(4.))
                        .corner_radius(CornerRadius::new_all(2.))
                        .background(colors.primary),
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
            SidebarView::Git => rect()
                .width(Size::fill())
                .height(Size::fill())
                .child(GitView {
                    state: self.state,
                    workspace: self.workspace,
                }),
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
        let mut state = self.state;
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
                    .height(Size::px(30.))
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .padding(Gaps::new(0., 6., 0., 8.))
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(11.)
                            .font_weight(FontWeight::MEDIUM)
                            .text("Explorer"),
                    )
                    .child(IconButton {
                        icon: "refresh",
                        label: "Refresh workspace",
                        size: 12.,
                        on_press: (move |_: Event<PressEventData>| workspace.refresh()).into(),
                    }),
            )
            .child(crate::ui::chrome::hairline())
            .child(
                ScrollView::new().direction(Direction::Vertical).child(
                    rect()
                        .width(Size::fill())
                        .vertical()
                        .on_secondary_down({
                            let workspace_root = state.read().workspace_root.clone();
                            move |event: Event<PressEventData>| {
                                if let PressEventData::Mouse(mouse) = event.data() {
                                    let mut app_state = state.write();
                                    app_state.explorer_focused = true;
                                    app_state.explorer_menu = Some(ExplorerMenu {
                                        path: workspace_root.clone(),
                                        is_dir: true,
                                        is_empty: true,
                                        x: mouse.global_location.x as f32,
                                        y: mouse.global_location.y as f32,
                                    });
                                }
                            }
                        })
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
            .height(Size::px(24.))
            .horizontal()
            .cross_align(Alignment::center())
            .spacing(6.)
            .padding(Gaps::new(0., 8., 0., 8.))
            .child(
                SvgViewer::new(("folder", icon("folder")))
                    .color(colors.text_secondary)
                    .width(Size::px(12.))
                    .height(Size::px(12.)),
            )
            .child(
                label()
                    .color(colors.text_primary)
                    .font_size(12.)
                    .font_weight(FontWeight::MEDIUM)
                    .max_lines(1)
                    .text(name),
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
            .height(Size::px(22.))
            .padding(0.)
            .on_secondary_down({
                let path = path.clone();
                move |event: Event<PressEventData>| {
                    event.stop_propagation();
                    if let PressEventData::Mouse(mouse) = event.data() {
                        let mut app_state = state.write();
                        app_state.explorer_focused = true;
                        app_state.explorer_menu = Some(ExplorerMenu {
                            path: path.clone(),
                            is_dir,
                            is_empty: false,
                            x: mouse.global_location.x as f32,
                            y: mouse.global_location.y as f32,
                        });
                    }
                }
            })
            .on_press(move |_| {
                let mut app_state = state.write();
                app_state.explorer_focused = true;
                app_state.selected_path = Some(path.clone());
                if is_dir {
                    crate::services::workspace::toggle_folder(&mut app_state.tree, &path);
                } else {
                    let already_open = app_state.activate_path(&path);
                    if !already_open {
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
                    .padding(Gaps::new(0., 6., 0., depth_padding))
                    .child(rect().width(Size::px(10.)).maybe(has_children, |rect| {
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
                            .width(Size::px(10.))
                            .height(Size::px(10.)),
                        )
                    }))
                    .child(
                        SvgViewer::new((file_icon(node.kind), icon(file_icon(node.kind))))
                            .color(colors.text_secondary)
                            .width(Size::px(12.))
                            .height(Size::px(12.)),
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
                    .height(Size::px(30.))
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .padding(Gaps::new(0., 8., 0., 8.))
                    .child(
                        rect()
                            .horizontal()
                            .cross_align(Alignment::center())
                            .spacing(6.)
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(11.)
                                    .font_weight(FontWeight::MEDIUM)
                                    .text("Changes"),
                            )
                            .child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(10.)
                                    .text(row_count.to_string()),
                            ),
                    ),
            )
            .child(crate::ui::chrome::hairline())
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
            .height(Size::px(26.))
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
            .on_secondary_down({
                let file_path = file.path.clone();
                move |event: Event<PressEventData>| {
                    event.stop_propagation();
                    let mut app_state = state.write();
                    let root = app_state.workspace_root.clone();
                    let full_path = root.join(&file_path);
                    if let Ok(disk_tab) = crate::services::document::open_file(&root, &full_path) {
                        let active_content = app_state
                            .tabs
                            .iter()
                            .find(|t| t.rel_path == file_path)
                            .map(|t| t.content.clone())
                            .unwrap_or_else(|| disk_tab.content.clone());
                        let diff = crate::services::diff::compute_line_diff(
                            &disk_tab.persisted_content,
                            &active_content,
                        );
                        app_state.diff_title = format!("Changes · {}", file_path);
                        app_state.diff_lines = diff.lines;
                        app_state.diff_open = true;
                    }
                }
            })
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::fill())
                    .horizontal()
                    .cross_align(Alignment::center())
                    .spacing(8.)
                    .padding(Gaps::new(0., 10., 0., 10.))
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
                                .text(file.path.clone()),
                        ),
                    )
                    .child(
                        Button::new()
                            .compact()
                            .flat()
                            .padding(Gaps::new(0., 5., 0., 5.))
                            .on_press({
                                let file_path = file.path.clone();
                                move |event: Event<PressEventData>| {
                                    event.stop_propagation();
                                    let mut app_state = state.write();
                                    let root = app_state.workspace_root.clone();
                                    let full_path = root.join(&file_path);
                                    if let Ok(disk_tab) =
                                        crate::services::document::open_file(&root, &full_path)
                                    {
                                        let active_content = app_state
                                            .tabs
                                            .iter()
                                            .find(|t| t.rel_path == file_path)
                                            .map(|t| t.content.clone())
                                            .unwrap_or_else(|| disk_tab.content.clone());
                                        let diff = crate::services::diff::compute_line_diff(
                                            &disk_tab.persisted_content,
                                            &active_content,
                                        );
                                        app_state.diff_title = format!("Changes · {}", file_path);
                                        app_state.diff_lines = diff.lines;
                                        app_state.diff_open = true;
                                    }
                                }
                            })
                            .child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(9.5)
                                    .text("diff"),
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
                    .width(Size::px(18.))
                    .height(Size::px(18.)),
            )
            .child(
                label()
                    .color(colors.text_primary)
                    .font_size(12.)
                    .font_weight(FontWeight::MEDIUM)
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

fn agent_status_color(colors: &ColorsSheet, status: AgentFileStatus) -> Color {
    match status {
        AgentFileStatus::Created => colors.success,
        AgentFileStatus::Modified => colors.warning,
        AgentFileStatus::Deleted => colors.error,
    }
}
