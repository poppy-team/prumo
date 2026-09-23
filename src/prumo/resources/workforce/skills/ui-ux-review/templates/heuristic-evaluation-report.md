# UI/UX Heuristic Evaluation Report: {{Product.Feature}}

## 1. Audit Metadata & Scope
- **Target Interface**: `{{interface_or_screen_name}}`
- **Evaluation Framework**: Nielsen Norman Group 10 Usability Heuristics
- **Evaluator**: `{{evaluator_agent_id}}`
- **Date**: {{YYYY-MM-DD}}
- **Overall Verdict**: [PASS | CONDITIONAL PASS | FAIL - RELEASE BLOCKER]

---

## 2. Executive Summary & Defect Metrics

| Severity Rating | Definition | Count |
| :---: | :--- | :---: |
| **Severity 4** | Usability Catastrophe (Release Blocker) | {{count_sev_4}} |
| **Severity 3** | Major Usability Problem (High Priority) | {{count_sev_3}} |
| **Severity 2** | Minor Usability Problem (Medium Priority) | {{count_sev_2}} |
| **Severity 1** | Cosmetic Problem (Low Priority) | {{count_sev_1}} |
| **Total** | | **{{total_defects}}** |

---

## 3. Cognitive Walkthrough Findings

| Step # | User Intent | Action Expected | Discovered Friction / Failure Mode |
| :---: | :--- | :--- | :--- |
| **01** | Locate Signup Form | Click "Get Started" | Prominent primary button in hero section [PASS] |
| **02** | Enter Password | Type valid credentials | Password requirements hidden until submit fails [FRICTION] |
| **03** | Confirm Account | Click verification link | Re-directs to login rather than dashboard [FRICTION] |

---

## 4. Heuristic Defect Log & Engineering Remediations

### Defect #1: [Brief Title]
- **Heuristic**: `H{{number}}: {{Heuristic Name}}`
- **Severity**: `[Severity 1 - 4]`
- **Location**: `{{Screen / Component Path / CSS Selector}}`
- **User Impact**: {{Detailed description of how the user experiences confusion, error, or blockage}}
- **Evidence / Screenshot Reference**: `{{asset_path_or_description}}`
- **Recommended Remediation**:
  {{Concrete engineering or design fix: prop change, copy update, state handler addition}}

---

## 5. Prioritized Engineering Action Items
1. **P0 (Immediate Blockers)**:
   - [ ] Fix Defect #X: {{description}}
2. **P1 (Next Sprint)**:
   - [ ] Fix Defect #Y: {{description}}
3. **P2 (Design Polish)**:
   - [ ] Fix Defect #Z: {{description}}
