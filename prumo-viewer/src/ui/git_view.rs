use crate::WorkspaceCommands;
use crate::services::git::{
    GitStatus, git_changed_lines, git_commit, git_discard, git_file_diff, git_pull, git_push,
    git_stage, git_status, git_unstage,
};
use crate::state::{AppState, FilePrompt, FilePromptKind, NoticeTone};
use crate::ui::chrome::IconButton;
use freya::code_editor::LineDecoration;
use freya::prelude::*;
use std::path::PathBuf;

#[derive(PartialEq)]
pub struct GitView {
    pub state: State<AppState>,
    pub workspace: WorkspaceCommands,
}

impl Component for GitView {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let state = self.state;
        let workspace = self.workspace;
        let git = state.read().git.clone();
        let message_state = use_state(String::new);

        rect()
            .width(Size::fill())
            .height(Size::fill())
            .background(colors.background)
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
                        rect()
                            .horizontal()
                            .cross_align(Alignment::center())
                            .spacing(6.)
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(11.)
                                    .font_weight(FontWeight::MEDIUM)
                                    .text("Git"),
                            )
                            .child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(10.)
                                    .max_lines(1)
                                    .text(git_branch_label(&git)),
                            ),
                    )
                    .child(
                        rect()
                            .horizontal()
                            .spacing(4.)
                            .cross_align(Alignment::center())
                            .child(
                                Button::new()
                                    .flat()
                                    .compact()
                                    .height(Size::px(22.))
                                    .padding(Gaps::new(0., 7., 0., 7.))
                                    .on_press(move |_| {
                                        run_git_op(state, "Push", |root| {
                                            git_push(root).map(|_| ())
                                        });
                                    })
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(10.)
                                            .text("Push"),
                                    ),
                            )
                            .child(
                                Button::new()
                                    .flat()
                                    .compact()
                                    .height(Size::px(22.))
                                    .padding(Gaps::new(0., 7., 0., 7.))
                                    .on_press(move |_| {
                                        run_git_op(state, "Pull", |root| {
                                            git_pull(root).map(|_| ())
                                        });
                                    })
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(10.)
                                            .text("Pull"),
                                    ),
                            )
                            .child(IconButton {
                                icon: "refresh",
                                label: "Refresh git status",
                                size: 12.,
                                on_press: (move |_: Event<PressEventData>| {
                                    refresh_git_status(state);
                                })
                                .into(),
                            }),
                    ),
            )
            .child(crate::ui::chrome::hairline())
            .child(if !state.read().git_ready {
                Element::from(
                    rect()
                        .width(Size::fill())
                        .height(Size::fill())
                        .horizontal()
                        .main_align(Alignment::center())
                        .child(
                            label()
                                .color(colors.text_secondary)
                                .font_size(11.)
                                .text("Git executable not found"),
                        ),
                )
            } else if !git.is_repo {
                Element::from(
                    rect()
                        .width(Size::fill())
                        .height(Size::fill())
                        .horizontal()
                        .main_align(Alignment::center())
                        .child(
                            label()
                                .color(colors.text_secondary)
                                .font_size(11.)
                                .text("Not a git repository"),
                        ),
                )
            } else {
                Element::from(
                    ScrollView::new()
                        .direction(torin::prelude::Direction::Vertical)
                        .height(Size::flex(1.))
                        .child(
                            rect()
                                .width(Size::fill())
                                .vertical()
                                .padding(Gaps::new(0., 4., 8., 4.))
                                .spacing(4.)
                                .child(
                                    rect()
                                        .width(Size::fill())
                                        .horizontal()
                                        .cross_align(Alignment::center())
                                        .spacing(4.)
                                        .child(
                                            rect().height(Size::px(24.)).font_size(11.).child(
                                                Input::new(message_state.into_writable())
                                                    .compact()
                                                    .placeholder("Commit message")
                                                    .width(Size::fill())
                                                    .on_submit({
                                                        let mut message_state = message_state;
                                                        move |value| {
                                                            submit_commit(state, value);
                                                            message_state.set(String::new());
                                                        }
                                                    }),
                                            ),
                                        )
                                        .child(
                                            Button::new()
                                                .compact()
                                                .height(Size::px(24.))
                                                .padding(Gaps::new(0., 10., 0., 10.))
                                                .enabled(
                                                    !message_state.read().trim().is_empty()
                                                        && git
                                                            .files
                                                            .iter()
                                                            .any(|file| file.staged_kind.is_some()),
                                                )
                                                .on_press({
                                                    let mut message_state = message_state;
                                                    move |_| {
                                                        submit_commit(
                                                            state,
                                                            message_state.read().clone(),
                                                        );
                                                        message_state.set(String::new());
                                                    }
                                                })
                                                .child(
                                                    label()
                                                        .color(colors.text_primary)
                                                        .font_size(10.5)
                                                        .text("Commit"),
                                                ),
                                        ),
                                )
                                .child(GitSection {
                                    state,
                                    workspace,
                                    title: "Staged",
                                    staged: true,
                                })
                                .child(GitSection {
                                    state,
                                    workspace,
                                    title: "Changes",
                                    staged: false,
                                }),
                        ),
                )
            })
    }
}

fn git_branch_label(git: &GitStatus) -> String {
    if !git.is_repo {
        return String::new();
    }
    let mut label = if git.branch.is_empty() {
        "detached".to_string()
    } else {
        git.branch.clone()
    };
    if git.ahead > 0 || git.behind > 0 {
        label.push_str(&format!(" ↑{} ↓{}", git.ahead, git.behind));
    }
    label
}

fn refresh_git_status(mut state: State<AppState>) {
    let root = state.read().workspace_root.clone();
    spawn(async move {
        let status = tokio::task::spawn_blocking(move || git_status(&root))
            .await
            .unwrap_or_default();
        state.write().git = status;
    });
}

fn run_git_op(
    state: State<AppState>,
    label: &'static str,
    op: impl FnOnce(&PathBuf) -> Result<(), String> + Send + 'static,
) {
    let root = state.read().workspace_root.clone();
    spawn(async move {
        let result =
            tokio::task::spawn_blocking(move || op(&root).map(|()| git_status(&root))).await;
        apply_git_status(state, label, result);
    });
}

fn apply_git_status(
    mut state: State<AppState>,
    label: &'static str,
    result: Result<Result<GitStatus, String>, tokio::task::JoinError>,
) {
    match result {
        Ok(Ok(status)) => {
            let mut app_state = state.write();
            app_state.git = status;
            let _ = crate::services::workspace::refresh_workspace(&mut app_state);
            app_state.show_notice(NoticeTone::Success, format!("Git {label} done"));
        }
        Ok(Err(error)) => state
            .write()
            .show_notice(NoticeTone::Error, format!("Git {label} failed: {error}")),
        Err(error) => state
            .write()
            .show_notice(NoticeTone::Error, format!("Git worker failed: {error}")),
    }
}

fn submit_commit(mut state: State<AppState>, message: String) {
    if message.trim().is_empty() {
        state
            .write()
            .show_notice(NoticeTone::Error, "Commit message must not be empty");
        return;
    }
    let root = state.read().workspace_root.clone();
    spawn(async move {
        let result = tokio::task::spawn_blocking(move || {
            git_commit(&root, &message).map(|_| git_status(&root))
        })
        .await;
        match result {
            Ok(Ok(status)) => {
                let mut app_state = state.write();
                app_state.git = status;
                app_state.show_notice(NoticeTone::Success, "Committed".to_string());
            }
            Ok(Err(error)) => state
                .write()
                .show_notice(NoticeTone::Error, format!("Commit failed: {error}")),
            Err(error) => state
                .write()
                .show_notice(NoticeTone::Error, format!("Git worker failed: {error}")),
        }
    });
}

#[derive(PartialEq)]
struct GitSection {
    state: State<AppState>,
    workspace: WorkspaceCommands,
    title: &'static str,
    staged: bool,
}

impl Component for GitSection {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let state = self.state;
        let workspace = self.workspace;
        let staged = self.staged;
        let files: Vec<_> = state
            .read()
            .git
            .files
            .iter()
            .filter(|file| {
                if staged {
                    file.staged_kind.is_some()
                } else {
                    file.unstaged_kind.is_some()
                }
            })
            .cloned()
            .collect();
        if files.is_empty() {
            return rect();
        }

        rect()
            .width(Size::fill())
            .vertical()
            .spacing(1.)
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(24.))
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .padding(Gaps::new(0., 0., 0., 8.))
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(10.)
                            .font_weight(FontWeight::MEDIUM)
                            .text(format!("{} ({})", self.title, files.len())),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(20.))
                            .padding(Gaps::new(0., 6., 0., 6.))
                            .on_press(move |_| {
                                let paths: Vec<String> = state
                                    .read()
                                    .git
                                    .files
                                    .iter()
                                    .filter(|file| {
                                        if staged {
                                            file.staged_kind.is_some()
                                        } else {
                                            file.unstaged_kind.is_some()
                                        }
                                    })
                                    .map(|file| file.path.clone())
                                    .collect();
                                let root = state.read().workspace_root.clone();
                                spawn(async move {
                                    let result = tokio::task::spawn_blocking(move || {
                                        for path in &paths {
                                            if staged {
                                                git_unstage(&root, path)?;
                                            } else {
                                                git_stage(&root, path)?;
                                            }
                                        }
                                        Ok::<_, String>(git_status(&root))
                                    })
                                    .await;
                                    apply_git_status(
                                        state,
                                        if staged { "Unstage all" } else { "Stage all" },
                                        result,
                                    );
                                });
                            })
                            .child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(10.)
                                    .text(if staged { "Unstage all" } else { "Stage all" }),
                            ),
                    ),
            )
            .children(files.into_iter().map(|file| {
                GitRow {
                    state,
                    workspace,
                    path: file.path.clone(),
                    marker: if staged {
                        file.staged_kind.unwrap_or('M')
                    } else {
                        file.unstaged_kind.unwrap_or('M')
                    },
                    staged,
                    tracked: file.unstaged_kind != Some('?') || staged,
                }
                .into()
            }))
    }
}

#[derive(PartialEq)]
struct GitRow {
    state: State<AppState>,
    workspace: WorkspaceCommands,
    path: String,
    marker: char,
    staged: bool,
    tracked: bool,
}

impl Component for GitRow {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let workspace = self.workspace;
        let path = self.path.clone();
        let staged = self.staged;
        let tracked = self.tracked;
        let marker = self.marker;
        let marker_color = match marker {
            'A' | '?' => colors.success,
            'D' => colors.error,
            _ => colors.warning,
        };
        let is_open = {
            let app_state = state.read();
            app_state
                .active_tab()
                .is_some_and(|tab| tab.path == app_state.workspace_root.join(&path))
        };

        rect()
            .width(Size::fill())
            .height(Size::px(26.))
            .background(if is_open {
                colors.surface_tertiary
            } else {
                Color::TRANSPARENT
            })
            .horizontal()
            .cross_align(Alignment::center())
            .spacing(4.)
            .padding(Gaps::new(0., 4., 0., 8.))
            .child(
                Button::new()
                    .flat()
                    .compact()
                    .width(Size::px(22.))
                    .height(Size::px(22.))
                    .padding(0.)
                    .on_press({
                        let path = path.clone();
                        move |_| {
                            let path = path.clone();
                            run_git_op(
                                state,
                                if staged { "Unstage" } else { "Stage" },
                                move |root| {
                                    if staged {
                                        git_unstage(root, &path)
                                    } else {
                                        git_stage(root, &path)
                                    }
                                },
                            );
                        }
                    })
                    .child(
                        label()
                            .color(marker_color)
                            .font_size(11.)
                            .font_weight(FontWeight::BOLD)
                            .text(self.marker.to_string()),
                    ),
            )
            .child(
                rect()
                    .width(Size::flex(1.))
                    .horizontal()
                    .cross_align(Alignment::center())
                    .on_press({
                        let path = path.clone();
                        move |event: Event<PressEventData>| {
                            event.stop_propagation();
                            open_git_file(state, workspace, path.clone(), staged);
                        }
                    })
                    .child(
                        label()
                            .color(colors.text_primary)
                            .font_size(11.5)
                            .max_lines(1)
                            .text(path.clone()),
                    ),
            )
            .child(IconButton {
                icon: "x",
                label: "Discard changes",
                size: 10.,
                on_press: (move |_: Event<PressEventData>| {
                    if tracked {
                        let path = path.clone();
                        run_git_op(state, "Discard", move |root| {
                            if staged {
                                git_unstage(root, &path)?;
                                if marker == 'A' {
                                    return git_discard(root, &path, false);
                                }
                            }
                            git_discard(root, &path, true)
                        });
                    } else {
                        state.write().file_prompt = Some(FilePrompt {
                            kind: FilePromptKind::ConfirmDelete(
                                state.read().workspace_root.join(&path),
                            ),
                            input: String::new(),
                        });
                    }
                })
                .into(),
            })
    }
}
fn open_git_file(
    mut state: State<AppState>,
    workspace: WorkspaceCommands,
    path: String,
    staged: bool,
) {
    let root = state.read().workspace_root.clone();
    let absolute_path = root.join(&path);
    if !absolute_path.is_file() {
        open_git_diff(state, path, staged);
        return;
    }
    spawn(async move {
        let result = tokio::task::spawn_blocking({
            let root = root.clone();
            let path = path.clone();
            let absolute_path = absolute_path.clone();
            move || {
                let lines = git_changed_lines(&root, &path, staged)?;
                let content = std::fs::read_to_string(&absolute_path)
                    .map_err(|error| format!("could not read Git file: {error}"))?;
                let first_line = lines.first().map(|(line, _)| (*line).max(1)).unwrap_or(1);
                let start = content
                    .split_inclusive('\n')
                    .take(first_line.saturating_sub(1))
                    .map(str::len)
                    .sum::<usize>();
                Ok::<_, String>((lines, first_line, start))
            }
        })
        .await;
        match result {
            Ok(Ok((changed, first_line, start))) => {
                let decorations = changed
                    .into_iter()
                    .map(|(line, tag)| LineDecoration {
                        line: line.saturating_sub(1),
                        color: match tag {
                            '+' => Color::from_argb(45, 46, 160, 67),
                            '~' => Color::from_argb(45, 220, 160, 60),
                            _ => Color::from_argb(45, 220, 90, 90),
                        },
                    })
                    .collect();
                {
                    let mut app_state = state.write();
                    app_state.pending_git_line_decorations = Some(decorations);
                    app_state.diff_open = false;
                }
                workspace.load_file_with_reveal(
                    absolute_path.clone(),
                    crate::state::EditorReveal {
                        path: absolute_path,
                        line: first_line,
                        start,
                        end: start,
                    },
                );
            }
            Ok(Err(error)) => state.write().show_notice(
                NoticeTone::Error,
                format!("Could not open Git file: {error}"),
            ),
            Err(error) => state.write().show_notice(
                NoticeTone::Error,
                format!("Git file worker failed: {error}"),
            ),
        }
    });
}

fn open_git_diff(mut state: State<AppState>, path: String, staged: bool) {
    let root = state.read().workspace_root.clone();
    spawn(async move {
        let result = tokio::task::spawn_blocking({
            let path = path.clone();
            move || git_file_diff(&root, &path, staged)
        })
        .await;
        match result {
            Ok(Ok(lines)) => {
                let mut app_state = state.write();
                app_state.diff_title = format!("Git diff · {path}");
                app_state.diff_lines = lines;
                app_state.diff_open = true;
            }
            Ok(Err(error)) => state
                .write()
                .show_notice(NoticeTone::Error, format!("Diff failed: {error}")),
            Err(error) => state
                .write()
                .show_notice(NoticeTone::Error, format!("Git worker failed: {error}")),
        }
    });
}
