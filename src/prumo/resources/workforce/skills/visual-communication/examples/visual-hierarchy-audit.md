# Example: Visual Hierarchy Audit (E-commerce Checkout)

## Overview
This audit evaluates the checkout page of `retail-demo-app` against visual communication principles, specifically focusing on visual hierarchy and scanning patterns.

## Current State Analysis

### 1. Hierarchy Levers
- **Size**: The promotional banner at the top of the page uses $32$px type, while the "Complete Purchase" button uses $14$px type.
  - *Critique*: Size lever is inverted. The primary action (purchasing) is dwarfed by secondary/tertiary information (promotions).
- **Color/Contrast**: The "Cancel Order" button and "Complete Purchase" button are both primary blue (`#0052CC`).
  - *Critique*: Lack of contrast differentiation creates ambiguity. Destructive/secondary actions carry the same visual weight as the primary conversion action.
- **Proximity**: Shipping address and billing address fields are equally spaced without a dividing visual boundary.
  - *Critique*: Gestalt grouping is weak. It requires cognitive effort to determine where one section ends and the other begins.

### 2. Semiotics
- The "Save for later" function is represented by a floppy disk icon.
  - *Critique*: The floppy disk is an archaic symbol that lacks a clear signifier-signified relationship for younger demographics.

### 3. Recommendations
1. **Fix Size/Scale**: Reduce the promotional banner text to $16$px. Increase the "Complete Purchase" button text to $18$px and increase the button's padding.
2. **Fix Color/Contrast**: Change the "Cancel Order" button to a ghost button (text-only or outline) using a neutral gray (`#6B778C`). Retain the primary blue for the checkout action to establish a clear hierarchy.
3. **Fix Proximity**: Introduce $32$px of vertical whitespace between the Shipping and Billing sections, and encapsulate them in subtle border-cards (`#DFE1E6`) to reinforce grouping.
4. **Fix Semiotics**: Replace the floppy disk icon with a heart symbol `❤️` or a bookmark symbol, which are contemporary, culturally established symbols for saving items.

## Tufte Principle Check
*N/A - No data visualizations on this page.*
