# Frontend Web Engineering Reference Guide

## 1. Core Concepts

### 1.1 Semantic HTML and the Accessibility Tree
Browsers build an accessibility tree from markup: a real button exposes role, name, and disabled state
for free, while a clickable div exposes nothing. Screen readers, voice control, and keyboard navigation
all consume this tree. Rule: if a native element exists for the job, use it; add ARIA only to fill gaps
native markup cannot express, following the first rule of ARIA.

### 1.2 Keyboard Interaction Model
Tab moves focus, Enter/Space activate, Escape dismisses, arrows navigate composite widgets. Focus must
always be visible and never lost: opening a modal moves focus inside, closing returns it to the trigger.
Roving tabindex manages arrow-key widgets like tablists and menus. Any flow completable by mouse must be
completable by keyboard at the same speed; test every release with keyboard only plus one screen reader
such as NVDA or VoiceOver.

### 1.3 Rendering and Cumulative Layout Shift
Cumulative Layout Shift (CLS) sums unexpected visual movement: CLS under 0.1 is good, above 0.25 is poor.
Largest Contentful Paint (LCP) under 2.5 s and Interaction to Next Paint (INP) under 200 ms complete the
Core Web Vitals triad. Reserve space for all media with width/height or aspect-ratio, never inject banners
or ads above content after load, and swap webfonts with font-display: swap to avoid invisible text.

### 1.4 Data Fetching States
Every remote read passes through idle, loading, success, empty, error, and offline states. SWR and React
Query model this with stale-while-revalidate: serve cached data instantly, revalidate in background every
30 s, retry failures with backoff. Optimistic writes update the UI immediately but must keep the previous
value for rollback and announce outcomes in an aria-live region.

### 1.5 Bundle Budgeting
JavaScript is the most expensive asset: parse and compile cost scales with size, especially on a Moto G54
class device. Budget 200 KB gzip for the initial route, split the rest by route and by heavy widget
(chart editors, rich text), and tree-shake icons and locales. Measure with the Vite bundle visualizer or
webpack-bundle-analyzer in CI and fail builds that exceed budget.

## 2. Patterns and Anti-Patterns

| Area | Pattern (do) | Anti-Pattern (avoid) |
|---|---|---|
| Controls | Native button, dialog, select | div with onClick and no role or tabindex |
| Focus | Visible ring, trap in modal, restore on close | outline: none with no replacement |
| Forms | Explicit labels, inline errors, aria-describedby | Placeholder-only inputs validated on blur alone |
| Live updates | Single aria-live region, concise messages | Assertive announcements on every keystroke |
| Images | AVIF/WebP, dimensions, lazy below fold | 4 MB hero JPEG shifting layout on load |
| State | Skeleton plus error with retry action | Blank screen or infinite spinner on failure |
| Fetching | SWR 30 s revalidate, 2 s timeout | Waterfall of six sequential useEffect fetches |

## 3. Code Example: Accessible Modal with Focus Management

```tsx
import { useEffect, useRef } from "react";

export function ConfirmDialog({ open, title, onClose, onConfirm }: {
  open: boolean; title: string; onClose: () => void; onConfirm: () => void;
}) {
  const dialogRef = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const el = dialogRef.current;
    if (!el) return;
    if (open && !el.open) el.showModal();
    if (!open && el.open) el.close();
  }, [open ]);

  useEffect(() => {
    const el = dialogRef.current;
    if (!el) return;
    const onCancel = (e: Event) => { e.preventDefault(); onClose(); };
    el.addEventListener("cancel", onCancel);
    return () => el.removeEventListener("cancel", onCancel);
  }, [onClose]);

  return (
    <dialog ref={dialogRef} aria-labelledby="confirm-title">
      <h2 id="confirm-title">{title}</h2>
      <p>This action cannot be undone.</p>
      <button onClick={onClose}>Keep order</button>
      <button onClick={onConfirm} autoFocus>Cancel order</button>
    </dialog>
  );
}
```

The native dialog element provides modal focus trapping, Escape handling, and top-layer rendering
without extra libraries; the cancel listener converts the native dismiss into the controlled onClose
path so focus restoration stays consistent.
