use crate::WorkspaceCommands;
use crate::services::files::{
    close_tabs_for_path, create_file, create_folder, delete_path, duplicate_path, move_to_trash,
    rename_path, retarget_tabs,
};
use crate::services::workspace::refresh_workspace;
use crate::state::{AppState, FilePrompt, FilePromptKind, NoticeTone};
use freya::prelude::*;

#[derive(Clone, Copy, PartialEq, Eq)]
enum ExplorerAction {
    NewFile,
    NewFolder,
    Rename,
    Duplicate,
    Delete,
}

const EXPLORER_ACTIONS: [(ExplorerAction, &str); 5] = [
    (ExplorerAction::NewFile, "New File"),
    (ExplorerAction::NewFolder, "New Folder"),
    (ExplorerAction::Rename, "Rename"),
    (ExplorerAction::Duplicate, "Duplicate"),
    (ExplorerAction::Delete, "Delete"),
];

#[derive(PartialEq)]
pub struct ExplorerMenuOverlay {
    pub state: State<AppState>,
}

impl Component for ExplorerMenuOverlay {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let menu = state.read().explorer_menu.clone();
        let Some(menu) = menu else {
            return rect();
        };
        let window = Platform::get().root_size.read();
        let x = menu.x.clamp(0., (window.width - 200.).max(0.));
        let y = menu.y.clamp(0., (window.height - 190.).max(0.));
        let menu_path = menu.path.clone();
        let menu_is_dir = menu.is_dir;
        let menu_is_empty = menu.is_empty;

        rect()
            .width(Size::px(window.width))
            .height(Size::px(window.height))
            .position(Position::new_global().top(0.).left(0.))
            .layer(Layer::Overlay)
            .background(Color::TRANSPARENT)
            .on_press(move |_| state.write().explorer_menu = None)
            .on_secondary_down(move |_| state.write().explorer_menu = None)
            .child(
                rect()
                    .width(Size::px(190.))
                    .vertical()
                    .spacing(2.)
                    .padding(Gaps::new_all(6.))
                    .position(Position::new_absolute().top(y).left(x))
                    .background(colors.surface_primary)
                    .border(Border::new().fill(colors.border).width(1.))
                    .corner_radius(CornerRadius::new_all(6.))
                    .on_mouse_up(|event: Event<MouseEventData>| {
                        event.stop_propagation();
                    })
                    .children(
                        EXPLORER_ACTIONS
                            .into_iter()
                            .filter(|(action, _)| {
                                !menu_is_empty
                                    || matches!(
                                        action,
                                        ExplorerAction::NewFile | ExplorerAction::NewFolder
                                    )
                            })
                            .map(|(action, text)| {
                                let is_delete = action == ExplorerAction::Delete;
                                let path = menu_path.clone();
                                Button::new()
                                    .flat()
                                    .expanded()
                                    .height(Size::px(28.))
                                    .padding(Gaps::new(0., 8., 0., 8.))
                                    .on_press(move |_| {
                                        let mut app_state = state.write();
                                        app_state.explorer_menu = None;
                                        let parent = if menu_is_dir {
                                            path.clone()
                                        } else {
                                            path.parent()
                                                .map(|parent| parent.to_path_buf())
                                                .unwrap_or_else(|| app_state.workspace_root.clone())
                                        };
                                        match action {
                                            ExplorerAction::NewFile => {
                                                app_state.file_prompt = Some(FilePrompt {
                                                    kind: FilePromptKind::NewFile(parent),
                                                    input: String::new(),
                                                });
                                            }
                                            ExplorerAction::NewFolder => {
                                                app_state.file_prompt = Some(FilePrompt {
                                                    kind: FilePromptKind::NewFolder(parent),
                                                    input: String::new(),
                                                });
                                            }
                                            ExplorerAction::Rename => {
                                                let initial = path
                                                    .file_name()
                                                    .and_then(|name| name.to_str())
                                                    .unwrap_or("")
                                                    .to_string();
                                                app_state.file_prompt = Some(FilePrompt {
                                                    kind: FilePromptKind::Rename(path.clone()),
                                                    input: initial,
                                                });
                                            }
                                            ExplorerAction::Duplicate => {
                                                match duplicate_path(&path) {
                                                    Ok(target) => {
                                                        let _ = refresh_workspace(&mut app_state);
                                                        app_state.show_notice(
                                                            NoticeTone::Success,
                                                            format!(
                                                                "Duplicated as {}",
                                                                target
                                                                    .file_name()
                                                                    .and_then(|name| name.to_str())
                                                                    .unwrap_or("file")
                                                            ),
                                                        );
                                                    }
                                                    Err(error) => app_state
                                                        .show_notice(NoticeTone::Error, error),
                                                }
                                            }
                                            ExplorerAction::Delete => {
                                                app_state.file_prompt = Some(FilePrompt {
                                                    kind: FilePromptKind::ConfirmTrash(
                                                        path.clone(),
                                                    ),
                                                    input: String::new(),
                                                });
                                            }
                                        }
                                    })
                                    .child(
                                        label()
                                            .color(if is_delete {
                                                colors.error
                                            } else {
                                                colors.text_primary
                                            })
                                            .font_size(11.5)
                                            .text(text),
                                    )
                                    .into()
                            }),
                    ),
            )
    }
}

#[derive(PartialEq)]
pub struct FilePromptModal {
    pub state: State<AppState>,
    pub workspace: WorkspaceCommands,
}

impl Component for FilePromptModal {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let workspace = self.workspace;
        let mut input_state = use_state(String::new);
        let mut active_prompt = use_state(|| None::<FilePrompt>);
        let prompt = state.read().file_prompt.clone();
        if *active_prompt.read() != prompt {
            active_prompt.set(prompt.clone());
            input_state.set(
                prompt
                    .as_ref()
                    .map(|item| item.input.clone())
                    .unwrap_or_default(),
            );
        }
        let Some(prompt) = prompt else {
            return rect();
        };
        let window = Platform::get().root_size.read();
        let (title, action_label, confirm) = match &prompt.kind {
            FilePromptKind::NewFile(_) => ("New File", "Create", true),
            FilePromptKind::NewFolder(_) => ("New Folder", "Create", true),
            FilePromptKind::Rename(_) => ("Rename", "Rename", true),
            FilePromptKind::ConfirmDelete(_) => ("Delete", "Delete", false),
            FilePromptKind::ConfirmTrash(_) => ("Move to Trash", "Move to Trash", false),
        };
        let is_trash = matches!(&prompt.kind, FilePromptKind::ConfirmTrash(_));
        let describe_target = match &prompt.kind {
            FilePromptKind::NewFile(parent) | FilePromptKind::NewFolder(parent) => parent
                .file_name()
                .and_then(|name| name.to_str())
                .unwrap_or("workspace")
                .to_string(),
            FilePromptKind::Rename(path)
            | FilePromptKind::ConfirmDelete(path)
            | FilePromptKind::ConfirmTrash(path) => path
                .file_name()
                .and_then(|name| name.to_str())
                .unwrap_or("item")
                .to_string(),
        };

        rect()
            .width(Size::px(window.width))
            .height(Size::px(window.height))
            .position(Position::new_global().top(0.).left(0.))
            .layer(Layer::Overlay)
            .background(Color::TRANSPARENT)
            .horizontal()
            .main_align(Alignment::center())
            .cross_align(Alignment::center())
            .on_press(move |_| state.write().file_prompt = None)
            .child(
                rect()
                    .width(Size::px(380.))
                    .vertical()
                    .spacing(10.)
                    .padding(Gaps::new_all(14.))
                    .background(colors.surface_primary)
                    .border(Border::new().fill(colors.border_focus).width(1.))
                    .corner_radius(CornerRadius::new_all(8.))
                    .on_mouse_up(|event: Event<MouseEventData>| {
                        event.stop_propagation();
                    })
                    .child(
                        label()
                            .color(colors.text_primary)
                            .font_size(13.)
                            .font_weight(FontWeight::MEDIUM)
                            .text(title),
                    )
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(11.)
                            .text(if confirm {
                                format!("Location: {describe_target}")
                            } else if is_trash {
                                format!("Move {describe_target} to the system trash?")
                            } else {
                                format!("Delete {describe_target} permanently?")
                            }),
                    )
                    .maybe(confirm, |el| {
                        el.child(
                            Input::new(input_state.into_writable())
                                .placeholder("Name")
                                .width(Size::fill())
                                .auto_focus(true)
                                .on_submit({
                                    let kind = prompt.kind.clone();
                                    move |value| {
                                        submit_prompt(state, workspace, kind.clone(), value);
                                    }
                                }),
                        )
                    })
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .main_align(Alignment::end())
                            .spacing(8.)
                            .child(
                                Button::new()
                                    .flat()
                                    .compact()
                                    .height(Size::px(26.))
                                    .padding(Gaps::new(0., 10., 0., 10.))
                                    .on_press(move |_| state.write().file_prompt = None)
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(11.)
                                            .text("Cancel"),
                                    ),
                            )
                            .child(
                                Button::new()
                                    .compact()
                                    .height(Size::px(26.))
                                    .padding(Gaps::new(0., 12., 0., 12.))
                                    .on_press({
                                        let kind = prompt.kind.clone();
                                        move |_| {
                                            let value = input_state.read().clone();
                                            submit_prompt(state, workspace, kind.clone(), value);
                                        }
                                    })
                                    .child(
                                        label()
                                            .color(if confirm {
                                                colors.text_primary
                                            } else {
                                                colors.error
                                            })
                                            .font_size(11.)
                                            .text(action_label),
                                    ),
                            ),
                    ),
            )
    }
}

fn submit_prompt(
    mut state: State<AppState>,
    workspace: WorkspaceCommands,
    kind: FilePromptKind,
    value: String,
) {
    let result = match &kind {
        FilePromptKind::NewFile(parent) => create_file(parent, &value).map(|path| {
            workspace.load_file(path.clone());
            format!(
                "Created {}",
                path.file_name()
                    .and_then(|name| name.to_str())
                    .unwrap_or("file")
            )
        }),
        FilePromptKind::NewFolder(parent) => create_folder(parent, &value).map(|path| {
            format!(
                "Created folder {}",
                path.file_name()
                    .and_then(|name| name.to_str())
                    .unwrap_or("folder")
            )
        }),
        FilePromptKind::Rename(path) => rename_path(path, &value).map(|next| {
            let mut app_state = state.write();
            retarget_tabs(&mut app_state, path, &next);
            format!(
                "Renamed to {}",
                next.file_name()
                    .and_then(|name| name.to_str())
                    .unwrap_or("file")
            )
        }),
        FilePromptKind::ConfirmDelete(path) => delete_path(path).map(|()| {
            let mut app_state = state.write();
            close_tabs_for_path(&mut app_state, path);
            app_state.git = crate::services::git::git_status(&app_state.workspace_root);
            "Deleted".to_string()
        }),
        FilePromptKind::ConfirmTrash(path) => {
            let root = state.read().workspace_root.clone();
            move_to_trash(&root, path).map(|()| {
                let mut app_state = state.write();
                close_tabs_for_path(&mut app_state, path);
                app_state.git = crate::services::git::git_status(&app_state.workspace_root);
                "Moved to system trash".to_string()
            })
        }
    };
    let mut app_state = state.write();
    match result {
        Ok(message) => {
            let _ = refresh_workspace(&mut app_state);
            app_state.file_prompt = None;
            app_state.show_notice(NoticeTone::Success, message);
        }
        Err(error) => app_state.show_notice(NoticeTone::Error, error),
    }
}
