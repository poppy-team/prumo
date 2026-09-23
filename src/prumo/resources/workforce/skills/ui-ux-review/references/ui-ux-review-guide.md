# UI/UX Heuristic Evaluation & Usability Engineering Guide

## 1. The Nielsen Norman 10 Usability Heuristics

Established by Jakob Nielsen and Rolf Molich, these ten principles form the gold standard for expert usability reviews:

### H1: Visibility of System Status
The design should always keep users informed about what is going on, through appropriate feedback within reasonable time.
- *Good*: Upload progress bar showing bytes transferred and estimated time remaining.
- *Bad*: Clicking "Submit" causes a 5-second frozen interface with no spinner or disabled button, prompting double clicks.

### H2: Match Between System and the Real World
The design should speak the users' language, using words, phrases, and concepts familiar to the user, rather than internal jargon.
- *Good*: "Save to Library" or "Move to Trash".
- *Bad*: "Persist entity to database cluster" or "Record 0x800412 purged".

### H3: User Control and Freedom
Users often perform actions by mistake. They need a clearly marked "emergency exit" to leave the unwanted action without having to go through an extended process.
- *Good*: "Undo" toast after archiving an email; explicit "Cancel" button in multi-step wizard.
- *Bad*: Modal dialog with no 'X' or cancel button, forcing form completion.

### H4: Consistency and Standards
Users should not have to wonder whether different words, situations, or actions mean the same thing. Follow platform and industry conventions (Jakob's Law: users spend most of their time on other sites).
- *Good*: Shopping cart icon in top right; search bar with magnifying glass.
- *Bad*: Using a floppy disk icon for "Download" and a tray icon for "Save".

### H5: Error Prevention
Even better than good error messages is a careful design which prevents a problem from occurring in the first place.
- *Good*: Datepicker preventing selection of past dates for flight departures; confirmation prompt before deleting a production database.
- *Bad*: Letting user fill out a 20-field form before alerting them that their username was already taken.

### H6: Recognition Rather Than Recall
Minimize the user's memory load by making elements, actions, and options visible. The user should not have to remember information from one part of the interface to another.
- *Good*: Displaying items in the checkout summary alongside payment fields.
- *Bad*: Requiring the user to remember a product SKU from screen 1 to type into screen 3.

### H7: Flexibility and Efficiency of Use
Shortcuts — hidden from novice users — may speed up the interaction for the expert user such that the design can cater to both inexperienced and experienced users.
- *Good*: Command palette (`Cmd+K`), keyboard shortcuts, customizable quick-action filters.
- *Bad*: Requiring 4 nested clicks to perform the most common daily operation.

### H8: Aesthetic and Minimalist Design
Interfaces should not contain information that is irrelevant or rarely needed. Every extra unit of information in an interface competes with the relevant units of information and diminishes their relative visibility.
- *Good*: Clean visual hierarchy with prominent primary action and subtle secondary actions.
- *Bad*: Dashboards cluttered with 40 distinct neon metric cards and equal visual weights.

### H9: Help Users Recognize, Diagnose, and Recover from Errors
Error messages should be expressed in plain language (no error codes), precisely indicate the problem, and constructively suggest a solution.
- *Good*: "The password must be at least 8 characters. Currently: 6 characters."
- *Bad*: "Error 500: Internal server assertion failure."

### H10: Help and Documentation
Even though it is better if the system can be used without documentation, it may be necessary to provide help and documentation.
- *Good*: Contextual tooltips on complex financial metrics; easily searchable onboarding guide.
- *Bad*: A 200-page unindexed PDF manual hidden behind a support link.

---

## 2. Usability Defect Severity Rating System

Severity is determined by combining three factors:
1. **Impact**: How severe is the damage or confusion?
2. **Frequency**: How common is the problem?
3. **Persistence**: Does the problem overcome the user once, or does it recur repeatedly?

| Rating | Classification | Operational Meaning | Action Required |
| :---: | :--- | :--- | :--- |
| **0** | Not a problem | Subjective preference or false alarm | No action required |
| **1** | Cosmetic only | Minor polish issue; doesn't impede flow | Fix if time permits |
| **2** | Minor usability | Low friction; users can work around it easily | Fix in regular sprint |
| **3** | Major usability | High friction; causes significant delay or error | High priority fix before launch |
| **4** | Catastrophe | Blocks task completion or risks critical data loss | Release blocker; immediate fix |
