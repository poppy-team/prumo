use crate::state::{AgentEventItem, AgentFileStatus, AgentStatus, ChangedFile};

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum QualitySeverity {
    Info,
    Warning,
    Failure,
}

impl QualitySeverity {
    pub fn label(self) -> &'static str {
        match self {
            Self::Info => "info",
            Self::Warning => "warning",
            Self::Failure => "failure",
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct QualityFinding {
    pub severity: QualitySeverity,
    pub code: String,
    pub summary: String,
    pub detail: Option<String>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct QualitySummary {
    pub failures: usize,
    pub warnings: usize,
    pub infos: usize,
}

impl QualitySummary {
    pub fn of(findings: &[QualityFinding]) -> Self {
        let mut summary = Self::default();
        for finding in findings {
            match finding.severity {
                QualitySeverity::Failure => summary.failures += 1,
                QualitySeverity::Warning => summary.warnings += 1,
                QualitySeverity::Info => summary.infos += 1,
            }
        }
        summary
    }

    pub fn is_clean(&self) -> bool {
        self.failures == 0 && self.warnings == 0
    }
}

pub fn findings(events: &[AgentEventItem]) -> Vec<QualityFinding> {
    events
        .iter()
        .filter_map(|event| match event.kind.as_str() {
            "tool.failed" => Some(QualityFinding {
                severity: QualitySeverity::Failure,
                code: "tool.failed".to_string(),
                summary: format!("Tool failed · {}", event.message),
                detail: None,
            }),
            "side_effect_refused" => Some(QualityFinding {
                severity: QualitySeverity::Failure,
                code: "side_effect.refused".to_string(),
                summary: format!("Side effect refused · {}", event.message),
                detail: None,
            }),
            "side_effect_journal_failed" => Some(QualityFinding {
                severity: QualitySeverity::Failure,
                code: "side_effect.journal".to_string(),
                summary: format!("Side effect journal failed · {}", event.message),
                detail: None,
            }),
            "scope_violation" => Some(QualityFinding {
                severity: QualitySeverity::Failure,
                code: "scope.violation".to_string(),
                summary: format!("Scope violation · {}", event.message),
                detail: None,
            }),
            "budget_exhausted" => Some(QualityFinding {
                severity: QualitySeverity::Failure,
                code: "budget.exhausted".to_string(),
                summary: format!("Budget exhausted · {}", event.message),
                detail: None,
            }),
            "side_effect_skipped" => Some(QualityFinding {
                severity: QualitySeverity::Warning,
                code: "side_effect.skipped".to_string(),
                summary: format!("Side effect skipped · {}", event.message),
                detail: None,
            }),
            "permission_denied" => Some(QualityFinding {
                severity: QualitySeverity::Warning,
                code: "permission.denied".to_string(),
                summary: format!("Permission denied · {}", event.message),
                detail: None,
            }),
            "run.paused" => Some(QualityFinding {
                severity: QualitySeverity::Warning,
                code: "run.paused".to_string(),
                summary: format!("Run paused · {}", event.message),
                detail: None,
            }),
            "run.finished" => run_finished_finding(&event.message),
            _ => None,
        })
        .collect()
}

fn run_finished_finding(message: &str) -> Option<QualityFinding> {
    let status = message
        .rsplit("· ")
        .next()
        .unwrap_or(message)
        .trim()
        .to_lowercase();
    match status.as_str() {
        "complete" | "completed" => Some(QualityFinding {
            severity: QualitySeverity::Info,
            code: "run.finished".to_string(),
            summary: "Run finished without failures".to_string(),
            detail: Some(message.to_string()),
        }),
        "failed" | "cancelled" => Some(QualityFinding {
            severity: QualitySeverity::Failure,
            code: "run.finished".to_string(),
            summary: format!("Run ended as {status}"),
            detail: Some(message.to_string()),
        }),
        _ => None,
    }
}

pub fn verdict(status: AgentStatus, summary: &QualitySummary) -> &'static str {
    match status {
        AgentStatus::Working | AgentStatus::AwaitingApproval if summary.is_clean() => {
            "Running with no findings so far"
        }
        AgentStatus::Working | AgentStatus::AwaitingApproval => "Running with findings",
        AgentStatus::Completed if summary.is_clean() => "Completed with no findings",
        AgentStatus::Completed => "Completed with findings",
        AgentStatus::Failed => "Run failed",
        AgentStatus::Disconnected => "Offline",
        AgentStatus::Idle => "Idle",
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum EvidenceKind {
    File,
    Tool,
    Approval,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EvidenceEntry {
    pub kind: EvidenceKind,
    pub title: String,
    pub detail: String,
    pub path: Option<String>,
}

pub fn evidence(events: &[AgentEventItem], changed_files: &[ChangedFile]) -> Vec<EvidenceEntry> {
    let mut entries: Vec<EvidenceEntry> = changed_files
        .iter()
        .map(|file| EvidenceEntry {
            kind: EvidenceKind::File,
            title: file.path.clone(),
            detail: match file.status {
                AgentFileStatus::Created => "created in this run".to_string(),
                AgentFileStatus::Modified => "modified in this run".to_string(),
                AgentFileStatus::Deleted => "deleted in this run".to_string(),
            },
            path: Some(file.path.clone()),
        })
        .collect();

    for event in events {
        match event.kind.as_str() {
            "tool_call_ready" => entries.push(EvidenceEntry {
                kind: EvidenceKind::Tool,
                title: event.message.clone(),
                detail: "tool call".to_string(),
                path: None,
            }),
            "permission_wait" => entries.push(EvidenceEntry {
                kind: EvidenceKind::Approval,
                title: event.message.clone(),
                detail: "awaiting approval".to_string(),
                path: None,
            }),
            _ => {}
        }
    }

    entries
}

#[cfg(test)]
mod tests {
    use super::*;

    fn event(kind: &str, message: &str) -> AgentEventItem {
        AgentEventItem {
            id: kind.to_string(),
            time: "10:00:00".to_string(),
            kind: kind.to_string(),
            message: message.to_string(),
        }
    }

    #[test]
    fn counts_failures_warnings_and_infos_from_real_events() {
        let events = vec![
            event("tool.failed", "Tool · test exited with 1"),
            event("side_effect_skipped", "already recorded as applied"),
            event("budget_exhausted", "cost limit"),
            event("text_delta", "noise"),
        ];

        let findings = findings(&events);
        let summary = QualitySummary::of(&findings);

        assert_eq!(summary.failures, 2);
        assert_eq!(summary.warnings, 1);
        assert!(!summary.is_clean());
    }

    #[test]
    fn a_finished_run_without_failures_is_informational() {
        let findings = findings(&[event("run.finished", "Run finished · complete")]);

        assert_eq!(findings.len(), 1);
        assert_eq!(findings[0].severity, QualitySeverity::Info);
    }

    #[test]
    fn a_failed_run_is_a_failure_with_its_reason() {
        let findings = findings(&[event("run.finished", "Run finished · failed")]);

        assert_eq!(findings[0].severity, QualitySeverity::Failure);
        assert!(findings[0].summary.contains("failed"));
    }

    #[test]
    fn says_the_run_is_clean_only_when_nothing_failed() {
        assert!(
            verdict(AgentStatus::Completed, &QualitySummary::default()).contains("no findings")
        );
        assert!(
            !verdict(
                AgentStatus::Completed,
                &QualitySummary {
                    failures: 1,
                    ..QualitySummary::default()
                }
            )
            .contains("no findings")
        );
    }

    #[test]
    fn evidence_lists_changed_files_before_tool_activity() {
        let events = vec![
            event("tool_call_ready", "Tool · edit"),
            event("permission_wait", "Approval required · bash"),
        ];
        let changed = vec![ChangedFile {
            path: "src/main.rs".to_string(),
            status: AgentFileStatus::Modified,
        }];

        let entries = evidence(&events, &changed);

        assert_eq!(entries[0].kind, EvidenceKind::File);
        assert_eq!(entries[0].path.as_deref(), Some("src/main.rs"));
        assert_eq!(entries[1].kind, EvidenceKind::Tool);
        assert_eq!(entries[2].kind, EvidenceKind::Approval);
    }
}
