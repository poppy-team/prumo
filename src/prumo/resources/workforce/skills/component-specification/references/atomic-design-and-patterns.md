# Reference: Atomic Design & Advanced Component Architecture

## 1. Atomic Design Hierarchy (Brad Frost)

Components are organized into five distinct levels of composition:

```
Atoms (Tokens & Primitives)
  └── Molecules (Functional Combinations)
        └── Organisms (Distinct Interface Sections)
              └── Templates (Page Layout Structures)
                    └── Pages (Real Content & State)
```

1. **Atoms:** Foundational UI elements that cannot be broken down further without losing utility:
   - `<Button>`, `<TextInput>`, `<Badge>`, `<Icon>`, `<Label>`, `<Avatar>`.
2. **Molecules:** Simple groups of UI atoms functioning together as a unit:
   - `<SearchBox>` (Input + Button + Search Icon).
   - `<FormField>` (Label + TextInput + ErrorMessage + HelpTooltip).
3. **Organisms:** Relatively complex, distinct components forming distinct sections of an interface:
   - `<NavBar>`, `<DataTable>`, `<HeroSection>`, `<SettingsDialog>`.
4. **Templates:** Page-level layout skeletons focusing on content structure and responsive container flow without concrete runtime data.
5. **Pages:** Specific template instances populated with real production data, user state, and localized copy.

---

## 2. Advanced Component Patterns

### 2.1 The Compound Component Pattern
Allows parent and child components to implicitly communicate state via React Context or Vue Provide/Inject while maintaining composable markup:

```tsx
<Select value={value} onValueChange={setValue}>
  <Select.Trigger aria-label="Framework">
    <Select.Value placeholder="Select a framework..." />
  </Select.Trigger>
  <Select.Content>
    <Select.Group>
      <Select.Item value="prumo">Prumo Framework</Select.Item>
      <Select.Item value="go">Go Toolchain</Select.Item>
    </Select.Group>
  </Select.Content>
</Select>
```

### 2.2 Class Variance Authority (CVA) & Variant API Design
Define variants strictly using type-safe discriminators:

```typescript
import { cva, type VariantProps } from 'class-variance-authority';

export const buttonVariants = cva(
  'inline-flex items-center justify-center font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'bg-brand text-white hover:bg-brand-emphasis active:bg-brand-active',
        destructive: 'bg-red-600 text-white hover:bg-red-700',
        outline: 'border border-border bg-transparent hover:bg-surface-hover',
        ghost: 'hover:bg-surface-hover text-text-primary',
        link: 'text-brand underline-offset-4 hover:underline'
      },
      size: {
        sm: 'h-8 px-3 text-xs rounded-md',
        md: 'h-10 px-4 text-sm rounded-lg',
        lg: 'h-12 px-6 text-base rounded-xl',
        icon: 'h-10 w-10 p-0 rounded-lg'
      }
    },
    defaultVariants: {
      variant: 'default',
      size: 'md'
    }
  }
);
```

---

## 3. Finite State Machine (FSM) Integration

Never model complex component state with multiple uncoordinated boolean flags (e.g. `isLoading`, `isError`, `isSuccess`). Use an explicit state union:

```typescript
type ComponentState =
  | { status: 'idle' }
  | { status: 'loading'; startedAt: number }
  | { status: 'success'; data: Payload }
  | { status: 'error'; error: ErrorDetail; canRetry: boolean };
```
