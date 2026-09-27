use crate::services::document::{
    file_uri_to_path, utf16_position_to_byte, utf16_position_to_line_character,
};
use freya::code_editor::CodeEditorData;
use prumo_extension_sdk::lsp::{
    LspCodeAction, LspCompletionItem, LspCompletionTextEdit, LspPosition, LspRange, LspTextEdit,
    LspWorkspaceEdit,
};
use serde_json::Value;
use std::cmp::Reverse;
use std::path::Path;

pub fn completion_text_edit(
    item: &LspCompletionItem,
    content: &str,
    cursor: usize,
) -> Result<LspTextEdit, String> {
    if item.insert_text_format == Some(2) {
        return Err("snippet completions are not supported yet".to_string());
    }
    if let Some(edit) = &item.text_edit {
        return Ok(match edit {
            LspCompletionTextEdit::Text(edit) => edit.clone(),
            LspCompletionTextEdit::InsertReplace(edit) => LspTextEdit {
                range: edit.replace.clone(),
                new_text: edit.new_text.clone(),
            },
        });
    }
    let (line, character) = utf16_position_to_line_character(content, cursor);
    let line = u32::try_from(line.saturating_sub(1))
        .map_err(|_| "completion line is outside the LSP range".to_string())?;
    let character = u32::try_from(character)
        .map_err(|_| "completion character is outside the LSP range".to_string())?;
    let position = LspPosition { line, character };
    Ok(LspTextEdit {
        range: LspRange {
            start: position.clone(),
            end: position,
        },
        new_text: item
            .insert_text
            .clone()
            .unwrap_or_else(|| item.label.clone()),
    })
}

pub fn apply_text_edits(
    editor: &mut CodeEditorData,
    content: &str,
    edits: &[LspTextEdit],
) -> Result<(), String> {
    let mut byte_edits = edits
        .iter()
        .map(|edit| {
            let start = utf16_position_to_byte(
                content,
                edit.range.start.line as usize + 1,
                edit.range.start.character as usize,
            )?;
            let end = utf16_position_to_byte(
                content,
                edit.range.end.line as usize + 1,
                edit.range.end.character as usize,
            )?;
            if start > end {
                return Err("LSP text edit has an inverted range".to_string());
            }
            Ok((start, end, edit.new_text.clone()))
        })
        .collect::<Result<Vec<_>, String>>()?;
    byte_edits.sort_by_key(|(start, _, _)| Reverse(*start));
    for pair in byte_edits.windows(2) {
        if pair[0].0 < pair[1].1 {
            return Err("overlapping LSP text edits are not supported".to_string());
        }
    }
    for (start, end, replacement) in byte_edits {
        editor.replace_byte_range(start, end, &replacement);
    }
    Ok(())
}

pub fn edits_for_path(edit: &LspWorkspaceEdit, path: &Path) -> Result<Vec<LspTextEdit>, String> {
    if edit
        .document_changes
        .as_ref()
        .is_some_and(|changes| !changes.is_null())
    {
        if edit
            .changes
            .as_ref()
            .is_some_and(|changes| !changes.is_empty())
        {
            return Err("workspace edit cannot combine changes and documentChanges".to_string());
        }
        let operations = edit
            .document_changes
            .as_ref()
            .and_then(Value::as_array)
            .ok_or_else(|| "workspace documentChanges must be an array".to_string())?;
        let mut active_edits = Vec::new();
        for operation in operations {
            let document = operation.get("textDocument").ok_or_else(|| {
                "workspace documentChanges contain an unsupported resource operation".to_string()
            })?;
            let uri = document
                .get("uri")
                .and_then(Value::as_str)
                .ok_or_else(|| "workspace textDocumentEdit has no URI".to_string())?;
            let candidate = file_uri_to_path(uri)
                .map_err(|error| format!("workspace edit contains an invalid URI: {error}"))?;
            if candidate != path {
                return Err("multi-file workspace edits are not supported yet".to_string());
            }
            let edits = operation
                .get("edits")
                .cloned()
                .ok_or_else(|| "workspace textDocumentEdit has no edits".to_string());
            active_edits.extend(
                serde_json::from_value::<Vec<LspTextEdit>>(edits?)
                    .map_err(|error| format!("workspace edit contains invalid edits: {error}"))?,
            );
        }
        if active_edits.is_empty() {
            return Err("workspace edit does not contain the active file".to_string());
        }
        return Ok(active_edits);
    }
    let changes = edit
        .changes
        .as_ref()
        .ok_or_else(|| "workspace edit has no changes map".to_string())?;
    let mut active_edits = None;
    for (uri, edits) in changes {
        let candidate = file_uri_to_path(uri)
            .map_err(|error| format!("workspace edit contains an invalid URI: {error}"))?;
        if candidate != path {
            return Err("multi-file workspace edits are not supported yet".to_string());
        }
        active_edits = Some(edits.clone());
    }
    active_edits.ok_or_else(|| "workspace edit does not contain the active file".to_string())
}

pub fn diagnostic_to_lsp(diagnostic: &crate::state::Diagnostic) -> serde_json::Value {
    let end_line = diagnostic.end_line.unwrap_or(diagnostic.line);
    let end_column = diagnostic.end_column.unwrap_or(diagnostic.column);
    serde_json::json!({
        "range": {
            "start": {
                "line": diagnostic.line.saturating_sub(1),
                "character": diagnostic.column.saturating_sub(1),
            },
            "end": {
                "line": end_line.saturating_sub(1),
                "character": end_column.saturating_sub(1),
            }
        },
        "severity": match diagnostic.severity {
            crate::state::DiagnosticSeverity::Error => 1,
            crate::state::DiagnosticSeverity::Warning => 2,
            crate::state::DiagnosticSeverity::Information => 3,
            crate::state::DiagnosticSeverity::Hint => 4,
        },
        "source": diagnostic.source,
        "code": diagnostic.code,
        "message": diagnostic.message,
    })
}

pub fn lsp_range_from_utf16(content: &str, range: (usize, usize)) -> Result<LspRange, String> {
    let (start_line, start_character) = utf16_position_to_line_character(content, range.0);
    let (end_line, end_character) = utf16_position_to_line_character(content, range.1);
    Ok(LspRange {
        start: LspPosition {
            line: u32::try_from(start_line.saturating_sub(1))
                .map_err(|_| "code action start line is outside the LSP range".to_string())?,
            character: u32::try_from(start_character)
                .map_err(|_| "code action start character is outside the LSP range".to_string())?,
        },
        end: LspPosition {
            line: u32::try_from(end_line.saturating_sub(1))
                .map_err(|_| "code action end line is outside the LSP range".to_string())?,
            character: u32::try_from(end_character)
                .map_err(|_| "code action end character is outside the LSP range".to_string())?,
        },
    })
}

pub fn code_action_edit(action: &LspCodeAction, path: &Path) -> Result<Vec<LspTextEdit>, String> {
    let edit = action
        .edit
        .as_ref()
        .ok_or_else(|| "code action has no workspace edit".to_string())?;
    let edits = edits_for_path(edit, path)?;
    if edits.is_empty() {
        return Err("code action contains no edits for the active file".to_string());
    }
    Ok(edits)
}

#[cfg(test)]
mod tests {
    use super::*;
    use freya::code_editor::{CodeEditorData, Rope};
    use prumo_extension_sdk::lsp::LspTextEdit;
    use std::path::PathBuf;

    fn edit(start: (u32, u32), end: (u32, u32), new_text: &str) -> LspTextEdit {
        LspTextEdit {
            range: LspRange {
                start: LspPosition {
                    line: start.0,
                    character: start.1,
                },
                end: LspPosition {
                    line: end.0,
                    character: end.1,
                },
            },
            new_text: new_text.to_string(),
        }
    }

    #[test]
    fn applies_non_overlapping_unicode_edits_in_reverse_order() {
        let content = "á foo foo";
        let mut editor = CodeEditorData::new(Rope::from_str(content), None);
        apply_text_edits(
            &mut editor,
            content,
            &[edit((0, 2), (0, 5), "bar"), edit((0, 6), (0, 9), "baz")],
        )
        .unwrap();
        assert_eq!(editor.rope.to_string(), "á bar baz");
        assert!(editor.undo());
        assert_eq!(editor.rope.to_string(), content);
    }

    #[test]
    fn rejects_overlapping_edits() {
        let content = "abcdef";
        let mut editor = CodeEditorData::new(Rope::from_str(content), None);
        let error = apply_text_edits(
            &mut editor,
            content,
            &[edit((0, 1), (0, 4), "x"), edit((0, 3), (0, 5), "y")],
        )
        .unwrap_err();
        assert!(error.contains("overlapping"));
        assert_eq!(editor.rope.to_string(), content);
    }

    #[test]
    fn builds_completion_edit_at_cursor() {
        let item = LspCompletionItem {
            label: "println".to_string(),
            kind: Some(3),
            detail: None,
            insert_text: None,
            insert_text_format: None,
            text_edit: None,
        };
        let edit = completion_text_edit(&item, "á", 1).unwrap();
        assert_eq!(edit.range.start.line, 0);
        assert_eq!(edit.range.start.character, 1);
        assert_eq!(edit.new_text, "println");
    }

    #[test]
    fn extracts_only_active_file_workspace_edits() {
        let path = PathBuf::from("/tmp/main.rs");
        let action = LspCodeAction {
            title: "Fix".to_string(),
            kind: Some("quickfix".to_string()),
            edit: Some(LspWorkspaceEdit {
                changes: Some(
                    [(
                        "file:///tmp/main.rs".to_string(),
                        vec![edit((0, 0), (0, 1), "m")],
                    )]
                    .into_iter()
                    .collect(),
                ),
                document_changes: None,
            }),
            command: None,
            is_preferred: None,
        };
        let edits = code_action_edit(&action, &path).unwrap();
        assert_eq!(edits[0].new_text, "m");
    }

    #[test]
    fn extracts_single_file_document_changes() {
        let path = PathBuf::from("/tmp/main.rs");
        let action = LspCodeAction {
            title: "Fix".to_string(),
            kind: None,
            edit: Some(LspWorkspaceEdit {
                changes: None,
                document_changes: Some(serde_json::json!([{
                    "textDocument": { "uri": "file:///tmp/main.rs", "version": 1 },
                    "edits": [{
                        "range": {
                            "start": { "line": 0, "character": 0 },
                            "end": { "line": 0, "character": 1 }
                        },
                        "newText": "m"
                    }]
                }])),
            }),
            command: None,
            is_preferred: None,
        };
        let edits = code_action_edit(&action, &path).unwrap();
        assert_eq!(edits[0].new_text, "m");
    }

    #[test]
    fn rejects_multi_file_workspace_edits() {
        let path = PathBuf::from("/tmp/main.rs");
        let action = LspCodeAction {
            title: "Fix".to_string(),
            kind: None,
            edit: Some(LspWorkspaceEdit {
                changes: Some(
                    [
                        (
                            "file:///tmp/main.rs".to_string(),
                            vec![edit((0, 0), (0, 1), "m")],
                        ),
                        (
                            "file:///tmp/other.rs".to_string(),
                            vec![edit((0, 0), (0, 1), "o")],
                        ),
                    ]
                    .into_iter()
                    .collect(),
                ),
                document_changes: None,
            }),
            command: None,
            is_preferred: None,
        };
        let error = code_action_edit(&action, &path).unwrap_err();
        assert!(error.contains("multi-file"));
    }
}
