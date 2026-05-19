# Unit NN: [Feature Name]

## Goal

One or two sentences describing the concrete, verifiable output of this unit.
Be specific. Bad: "Create the auth pages." Good: "Create sign-in and sign-up
pages using Clerk components with a two-panel layout on desktop and form-only
on mobile. Use proxy.ts for route protection, not middleware.ts."

## Design

Visual and structural decisions specific to this unit. Reference
`context/ui-context.md` tokens by name. Describe layout behavior, component
choices, responsiveness. The agent should make zero visual decisions on its own.

## Implementation

### [Component or sub-section name]

Detailed description of what to build, including file paths, props, state,
and behavior. Enough detail that there is no ambiguity about what done looks like.

### [Next sub-section]

Description.

## Dependencies

- `package-name` — reason it's needed in this unit
- (or: none)

## Verify when done

- [ ] Condition one
- [ ] Condition two
- [ ] No TypeScript errors
- [ ] No console errors
- [ ] Responsive at mobile and desktop
- [ ] `npm run build` passes
