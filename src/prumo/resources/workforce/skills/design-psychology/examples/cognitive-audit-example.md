# Cognitive Audit Report: Acme E-Commerce Checkout Flow

**Date:** 2026-09-27  
**Auditor:** UX Research Agent  
**Target User Profile:** General consumers, varying technical proficiency, high task-oriented intent (purchasing).

## 1. Executive Summary
The Acme checkout flow currently suffers from high extraneous cognitive load and Fitts's Law violations on mobile devices. By reorganizing the form fields using Miller's Law and optimizing CTA placement, we can reduce cart abandonment.

## 2. Hick's Law & Decision Complexity
- **Current State:** At the shipping method selection, users are presented with 8 different shipping speeds, all presented as a flat list with small text.
- **Calculation/Estimation:** High decision time due to $n=8$ unorganized options.
- **Recommendation:** Implement progressive disclosure. Group shipping into "Standard", "Expedited", and "Overnight", then allow users to select specific carriers within those 3 categories.

## 3. Fitts's Law & Target Acquisition
- **Current State:** The "Place Order" button on mobile is $120 \times 32$px and placed centrally, requiring the user to stretch their thumb.
- **Recommendation:** Increase touch target to full width ($100\%$ of container) with a minimum height of $48$px. Anchor it to the bottom edge of the viewport.

## 4. Gestalt Principles Violations
| Element/Section | Gestalt Principle Violated | Description of Friction | Proposed Fix |
| :--- | :--- | :--- | :--- |
| Billing Address | Proximity | The checkbox for "Billing same as Shipping" is too far below the shipping address, disconnecting the two concepts. | Move the checkbox immediately below the Shipping Address block. |
| Form Fields | Figure-Ground | Input field borders are very light gray (#EEEEEE) against a white background, making them hard to see. | Darken borders to #999999 to increase contrast and define the input area. |

## 5. Cognitive Load Analysis
- **Extraneous Load Identified:** The sidebar contains promotional banners and links to other products during the checkout phase.
- **Miller's Law Compliance:** The single 15-field checkout form overwhelms working memory.
- **Recommendation for Load Reduction:** Remove sidebar promotions (isolate the checkout flow). Break the 15-field form into a 3-step wizard (Account, Shipping, Payment) to chunk the information.

## 6. Attention & Memory Models
- **Von Restorff Effect:** The "Place Order" button uses the same blue color as secondary buttons (e.g., "Apply Promo Code"). Change "Place Order" to a high-contrast accent color (e.g., green/orange).
- **Serial Position Effect:** Order summary details are hidden in an accordion. Surface the total price and expected delivery date clearly at the end of the flow.
- **Doherty Threshold:** Credit card processing takes ~2 seconds. Add a skeleton screen or an engaging loading animation instead of a static spinner.

## 7. Action Items
1. [x] Change "Place Order" button styling (Fitts's Law, Von Restorff).
2. [ ] Refactor form into a 3-step wizard (Miller's Law).
3. [ ] Categorize shipping options (Hick's Law).
4. [ ] Remove extraneous sidebar content (Cognitive Load Theory).
