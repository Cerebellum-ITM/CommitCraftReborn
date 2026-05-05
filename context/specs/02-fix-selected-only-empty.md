# Unit 02: fix-selected-only-empty

## Goal

After the user marks N commits with space in `stateReleaseChoosingCommits`
and toggles `ctrl+e` to "Selected only", the workspace commit list must
show exactly those N commits — no more, no less. Today the user reports
the visible list is empty even when prior selections exist
(`progress-tracker.md` audit, P2).

This unit reproduces the bug, identifies the root cause, fixes it, and
adds enough defense (tests / asserts) that the same regression doesn't
silently come back.

## Design

No visual changes. The "All commits / Selected only" segmented pill
stays as-is — same labels, same `^E` hint, same `theme.AppStyles().Help`
rendering. Only the data path behind the toggle changes.

## Implementation

### Step 1 — Reproduce and instrument

The wiring **looks correct** on paper:

- `applyReleaseChooseModeFilter` (`update_release.go:661`) sets
  `releaseChooseSelectedOnly=true`, hands a non-empty sentinel to
  `SetFilterText`, then forces `SetFilterState(FilterApplied)`.
- `WorkspaceCommitItem.FilterValue` (`release_list.go:63-80`) returns
  `""` for unselected items when the flag is on.
- `releaseChooseListFilter` (`release_list.go:104-132`) recognises the
  sentinel, drops empty `FilterValue`s, and returns the kept indices.
- `bubbles/v2/list.SetFilterText` does call `filterItems` synchronously
  and assigns `m.filteredItems` (verified in
  `~/go/pkg/mod/charm.land/bubbles/v2@v2.1.0/list/list.go:280-291`).

Since the bug exists despite this, the implementation must start by
**actually reproducing it** before changing code. Two paths:

1. Build (`go build ./...`), launch `commitcraft -r` against a repo with
   ≥ 3 commits, mark 2 with space, press `ctrl+e`, observe.
2. If the bug doesn't reproduce on a fresh launch, try the sequence
   reported in the audit: type something into the filter bar first,
   clear it, then mark commits, then `ctrl+e`. Filter-bar state may
   leak.

Add temporary debug logging (`internal/logger`) inside
`applyReleaseChooseModeFilter` to dump `selectedOnly`, `val`,
`filterText`, the count of items where `Selected==true`, and the count
the bubble reports for `releaseCommitList.VisibleItems()` *after*
`SetFilterText`. Remove the logging before commit.

### Step 2 — Suspects to rule out, in order

Investigate these in order — most-likely first. The fix lives wherever
the first one bites.

1. **`Index()` in filtered space corrupts the selection toggle.**
   `update_release.go:609-617` reads
   `index := model.releaseCommitList.Index()` and writes
   `SetItem(index, item)`. When the list is in `FilterApplied` state,
   `Index()` is the cursor position in the filtered view, but
   `SetItem(idx, …)` writes to `m.items[idx]` (the underlying slice,
   per `bubbles/v2/list/list.go:411-413`). After a prior `ctrl+e`
   round-trip, the user can end up flipping `Selected` on the *wrong*
   underlying commit, producing a phantom "selected" set that
   `releaseChooseSelectedOnly` then can't see.
   **Fix:** when `filterState != Unfiltered`, translate the visible
   cursor index back to the underlying-items index before calling
   `SetItem`. Use `releaseCommitList.VisibleItems()` and match by
   `WorkspaceCommitItem.Hash`, or read the filtered match list
   directly. Document the mapping in a comment so the next reader
   sees why the indirection exists.

2. **`releaseChooseModeBar` initial mode mismatch.** `Toggle()`
   (`history_mode_bar.go:52`) flips between `ModeKeyPointsBody` and
   `ModeStagesResponse`. Default is `ModeKeyPointsBody`. If anything
   ever calls `SetMode(ModeStagesResponse)` at construction (e.g. a
   future preference), the visual pill highlights "Selected only" but
   the first user `ctrl+e` flips it to "All commits" — and if the user
   reads the visual instead of pressing twice, they think they're in
   selected-only mode while logically they're in all-commits. Confirm
   the initial mode at startup with a one-line debug log. **Fix:** if
   mismatch confirmed, lock the initial mode to `ModeKeyPointsBody` in
   `Model` constructor (`model.go:448-449`).

3. **Filter-bar value carries a stale sentinel-like string.** If a
   previous code path called `SetFilterText(releaseChooseSentinel)` and
   then the user typed normally, `releaseChooseFilterBar.Value()` could
   theoretically still hold residue. Unlikely — `Reset()` sets `""` —
   but worth one print to confirm `val == ""` at the moment of `ctrl+e`.

4. **`m.items` does not reflect the most recent toggle.** `SetItem`
   writes to `m.items` synchronously (`list.go:412`). But the toggle
   handler reads `item, _ := releaseCommitList.SelectedItem().(WorkspaceCommitItem)`
   then mutates the local copy — `item.Selected = true` —
   then `SetItem(index, item)`. Confirm via the debug log that
   `m.items[i].(WorkspaceCommitItem).Selected` is actually `true` for
   the items the user marked, *immediately before* `ctrl+e` runs the
   filter. If they're all `false`, the toggle path is the bug, not the
   filter path.

### Step 3 — Apply the fix

Once the root cause is confirmed, change only the file(s) implicated.
The most likely change set is:

- `internal/tui/update_release.go` — at the `key.Matches(msg, model.keys.AddCommit)`
  branch (around line 590), translate the cursor index from the
  filtered view to the underlying items slice before `SetItem`. Pseudo-shape:

      idx := model.releaseCommitList.Index()
      if model.releaseCommitList.FilterState() != list.Unfiltered {
          // Cursor is in the filtered slice; remap to m.items by hash.
          visible := model.releaseCommitList.VisibleItems()
          if idx >= 0 && idx < len(visible) {
              if vis, ok := visible[idx].(WorkspaceCommitItem); ok {
                  for j, raw := range model.releaseCommitList.Items() {
                      if w, ok := raw.(WorkspaceCommitItem); ok && w.Hash == vis.Hash {
                          idx = j
                          break
                      }
                  }
              }
          }
      }
      cmd = model.releaseCommitList.SetItem(idx, item)

- Update the existing comment block at `update_release.go:611-616` so
  the rationale for the new indirection is explicit.

If the bug is suspect 2 (mode-bar default), change is in `model.go`
constructor only. If suspect 4 (toggle handler doesn't actually mutate
`m.items`), the fix may be to swap `SetItem` for a struct-pointer
pattern, but that's a bigger refactor — confirm with the user before
expanding scope.

### Step 4 — Defensive assertions

Add a single guarded log line in `applyReleaseChooseModeFilter`:

    if selectedOnly && len(model.selectedCommitList) > 0 &&
        len(model.releaseCommitList.VisibleItems()) == 0 {
        logger.Get().Warn("selected-only filter produced empty visible set despite selectedCommitList having entries",
            "selected", len(model.selectedCommitList))
    }

This is a permanent canary — if the same shape regresses, the user sees
it in `commitcraft -r ... ; less ~/.config/CommitCraft/commitcraft.log`
without filing another bug.

### Step 5 — CHANGELOG + version bump

- `cmd/cli/main.go`: bump `v0.49.0` → `v0.49.1` (patch — bug fix, no
  user-visible new surface).
- `CHANGELOG.md`: new top entry under `## v0.49.1 — <date>`. One
  paragraph: what was broken, what triggers the fix, no need for a
  `### Usage` block (no new feature surface).

## Dependencies

- none.

## Verify when done

- [ ] Manual: launch `commitcraft -r` against a repo with ≥ 3 commits;
      mark exactly 2 with space; press `ctrl+e`; the visible list
      contains exactly those 2 commits, in their original order, with
      the cursor on the first.
- [ ] Manual: in "Selected only" mode, press space on a visible row;
      that commit unselects, the visible set drops to 1 commit, the
      cursor stays on a real row (not row 0 if it wasn't there before).
- [ ] Manual: switch back to "All commits" with `ctrl+e`; both visible
      and underlying ordering restored, the previously-selected
      remaining commit still has its tick glyph.
- [ ] Manual: type a fuzzy filter while in "Selected only" mode; only
      selected commits matching the term remain visible.
- [ ] Manual: clear the filter bar with esc; "Selected only" stays
      active; visible set returns to selected-only without flicker.
- [ ] `grep -n "VisibleItems\|FilterState" internal/tui/update_release.go`
      shows the new index-translation block at the toggle-selection
      handler.
- [ ] Temporary debug logging from Step 1 is removed; only the Step 4
      defensive canary remains.
- [ ] `go build ./...` and `go vet ./...` pass.
- [ ] `cmd/cli/main.go` version bumped; `CHANGELOG.md` has a new entry
      at the top in English.
- [ ] Pre-commit hook (`gofumpt → goimports-reviser → golines`) passes.
