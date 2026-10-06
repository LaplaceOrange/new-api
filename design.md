---
name: new-api-console-design
description: Presentation rules for new-api administrative lists, detail dashboards, and compact configuration dialogs in light and dark themes.
version: 2026-10-06
---

# 1. Scope and Priority

[MUST] Preserve new-api and QuantumNous identity and attribution.
[MUST] Apply these rules to administrative list, detail, and dialog surfaces.
[SHOULD] Leave marketing surfaces and unrelated existing pages unchanged.
[MUST] Resolve conflicts in this order: facts, accessibility, user requirements,
project conventions, visual expression, decoration.

# 2. Brand and Readers

[SHOULD] Support expert readers with compact controls and aligned numeric values.
[SHOULD] Use short operational labels and explicit failure states.
[SHOULD] Inherit the selected theme rather than introducing a separate palette.

# 3. Page Structure and Composition

[SHOULD] Place the page heading and primary actions on one aligned header band.
[SHOULD] Place filters immediately above repeated content.
[SHOULD] Use the shared table/card surface for dense repeated entries.
[SHOULD] Keep detail sections unframed, separated by spacing and rules.
[SHOULD] Stack secondary regions below primary content on narrow screens.
[SHOULD] Keep dialog actions outside the scrolling form body.

# 4. Visual Rules

[MUST] Use the host body font through `--font-body`.
[SHOULD] Use `text-sm` for reading text and `text-xs` for secondary metadata.
[SHOULD] Use `text-base font-semibold` for repeated item headings.
[SHOULD] Use `text-2xl font-semibold tabular-nums` for primary amounts.
Decision: large amounts must remain scannable in compact repeated entries;
source: requested information hierarchy and existing `StatCard`.
[MUST] Keep letter spacing at zero on new surfaces.
[SHOULD] Use `background`, `foreground`, `muted-foreground`, and `border` tokens.
[MUST] Pair status colors with visible text.
[SHOULD] Use destructive, warning, and success tokens for semantic feedback.
[SHOULD] Distinguish intermediate caution with amber text and a subtle yellow
marker, not a saturated page background.
Decision: source is the requested four-level status presentation.
[SHOULD] Use a single subdued border on repeated items; do not frame sections.
[MUST] Keep new repeated-item corners at 8px or less.
[SHOULD] Use spacing utilities `gap-2`, `gap-3`, `gap-4`, `p-3`, and `p-4`.
[SHOULD] Reflow toolbars below 640px. Use one content column below 768px,
two from 768px, and three from 1024px.
Decision: source is the existing `DataTableCardGrid` responsive structure.
[MUST] Prevent document-level horizontal overflow.
[MUST] Let long identifiers wrap or truncate with an accessible full value.
[MUST] Align numeric table headings with their cells.
[MUST] Retain visible focus indicators and accessible icon names.
[SHOULD] Use skeletons while loading and shared error/empty states on failure.
[MUST] Mark stale or incomplete values without implying current precision.
[SHOULD] Retain host reduced-motion behavior; do not add decorative motion.

# 5. Available Primitives

| Role | Implementation | Source | Usage | Status |
|---|---|---|---|---|
| Body font | `--font-body` | `web/src/styles/theme.css` | Reading text | Implemented |
| Semantic colors | `background`, `foreground`, `muted-foreground`, `border`, `destructive`, `warning`, `success` | `web/src/styles/theme.css` | Host themes | Implemented |
| Page layout | `SectionPageLayout` | `web/src/components/layout` | Lists and details | Implemented |
| List surface | `DataTablePage`, `useDataTable` | `web/src/components/data-table` | Repeated entries | Implemented |
| View selector | `DataTableViewModeToggle` | `web/src/components/data-table` | Table/card modes | Implemented |
| Dialog | `Dialog` with `title`, `footer` | `web/src/components/dialog.tsx` | Configuration forms | Implemented |
| Confirmation | `ConfirmDialog` | `web/src/components/confirm-dialog.tsx` | Destructive actions | Implemented |
| Feedback | `EmptyState`, `LoadingState`, `ErrorState` | `web/src/components` | Data states | Implemented |
| Controls | `Button`, `Input`, `Textarea`, `Select`, `Switch`, `Tabs`, `ToggleGroup`, `Label` | `web/src/components/ui` | Native control roles | Implemented |
| Form rows | `FieldGroup`, `Field`, `FieldLabel`, `FieldError`, `FieldDescription` | `web/src/components/ui/field.tsx` | Labelled inputs and validation states | Implemented |
| Masked input | `PasswordInput` | `web/src/components/password-input.tsx` | Masked values with an explicit visibility action | Implemented |
| Icons | Installed `lucide-react` icons | `web/package.json` | Commands and status | Implemented |
| Verification | `SecureVerificationDialog` | `web/src/features/auth/secure-verification` | Sensitive actions | Implemented |

[MUST] Use public exports of these primitives; verify real props before use.
[SHOULD] Page-specific styling has no naming namespace limit.
[MUST] Do not replace published component behavior with page-local duplicates.
[SHOULD] Compose primitives without changing their global typography or surfaces.

# 6. Copy and Number Formats

[SHOULD] Display amounts with a currency symbol and two decimal places.
[SHOULD] Display durations with a unit and at most two decimal places.
[MUST] Label estimates as estimates.
[MUST] Show the time zone with timestamp ranges.
[MUST] Do not expose credentials, exception objects, or implementation diagnostics.
[SHOULD] Describe failures with the failed operation and a recovery action.

# 7. Anti-Patterns

[SHOULD] Do not use a centered hero for administrative surfaces.
[MUST] Do not put cards inside cards.
[SHOULD] Do not add decorative icon tiles or colored icon backgrounds.
[SHOULD] Do not introduce parallel fonts or global color literals.
[MUST] Do not render primary information as small, low-contrast metadata.
[SHOULD] Do not use pill labels for ordinary metadata.

# 8. Implementation and Integration

[SHOULD] Use `web/src/styles/index.css` and `theme.css` as style authorities.
[SHOULD] Keep existing font loading and theme switching intact.
[SHOULD] Source controls and layouts from the listed project components.

# Glossary

| Concept | Name |
|---|---|
| Page layout | `SectionPageLayout` |
| Repeated content surface | `DataTablePage` |
| Compact configuration overlay | `Dialog` |
| Destructive confirmation | `ConfirmDialog` |
| Semantic muted text | `muted-foreground` |
| Stable numeric alignment | `tabular-nums` |
