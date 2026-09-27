# Typography System Validation Checklist

- [ ] Base body size is defined at 16px (1rem) or greater.
- [ ] Modular scale ratio is mathematically consistent (e.g., 1.125, 1.25).
- [ ] Fluid typography values use `clamp(min, preferred, max)`.
- [ ] Line lengths for reading text are restricted to 45-75 `ch`.
- [ ] OpenType features (e.g., `kern`, `tnum` for data tables) are enabled where appropriate.
- [ ] Variable font axes (`wght`, `slnt`, etc.) are correctly mapped to tokens.
- [ ] Vertical rhythm strictly follows the base 4px or 8px baseline grid.
- [ ] Tokens are structured according to DTCG (Design Token Community Group) specifications.
- [ ] Typography scale gracefully handles 200% zoom for accessibility.
