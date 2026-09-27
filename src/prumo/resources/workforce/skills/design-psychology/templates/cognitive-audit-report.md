# Cognitive Audit Report: [Project/Interface Name]

**Date:** [YYYY-MM-DD]  
**Auditor:** [Name/Role]  
**Target User Profile:** [Brief description of the user]

## 1. Executive Summary
[Brief summary of the overall cognitive load and usability of the interface. Highlight the most critical psychological friction points.]

## 2. Hick's Law & Decision Complexity
- **Current State:** [Describe the number of choices, e.g., "The main navigation contains 24 flat links."]
- **Calculation/Estimation:** $T = a + b \log_2(24+1)$ indicates high cognitive friction.
- **Recommendation:** [e.g., "Implement progressive disclosure by grouping links into 4 primary categories."]

## 3. Fitts's Law & Target Acquisition
- **Current State:** [e.g., "The 'Submit' button on mobile is $24 \times 12$px and located in the center-middle."]
- **Recommendation:** [e.g., "Increase touch target to $48 \times 48$px and move to the bottom edge for thumb reachability."]

## 4. Gestalt Principles Violations
| Element/Section | Gestalt Principle Violated | Description of Friction | Proposed Fix |
| :--- | :--- | :--- | :--- |
| [e.g., Pricing Table] | Proximity | Feature lists are equally spaced between plan columns, making it hard to see which features belong to which plan. | Increase margin between columns; decrease line-height within lists. |
| | | | |

## 5. Cognitive Load Analysis
- **Extraneous Load Identified:** [List elements that distract without adding value]
- **Miller's Law Compliance:** [Are items chunked properly?]
- **Recommendation for Load Reduction:** [Steps to simplify]

## 6. Attention & Memory Models
- **Von Restorff Effect:** [Is the primary CTA distinct? If not, how to fix it.]
- **Serial Position Effect:** [Are key items at the start/end?]
- **Doherty Threshold:** [Are there long wait times >400ms without feedback? Recommend skeleton screens.]

## 7. Action Items
1. [ ] Action item 1
2. [ ] Action item 2
3. [ ] Action item 3
