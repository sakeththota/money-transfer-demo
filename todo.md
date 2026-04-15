# Code Convention TODOs

## Go Conventions

- [ ] **Module name**: Rename `money-transfer-worker` to `money-transfer-demo` or a proper import path like `github.com/user/money-transfer` (reflects whole project, not just worker)

- [ ] **Package `app`**: Rename to `transfer` or merge structs into `workflows` package (name packages by what they provide, not generic containers)

- [ ] **SCREAMING_CASE constants**: Change to Go-style CamelCase
  - `ADVANCED_VISIBILTY` → `AdvancedVisibility` (also fix typo - missing "I")
  - `NEEDS_APPROVAL` → `NeedsApproval`
  - `BUG` → `Bug`
  - `APPROVAL_TIME` → `ApprovalTime`

- [ ] **Shadowed builtin**: In `activities/deposit.go`, `error` variable shadows builtin type. Rename to `err`

- [ ] **Function signature**: `simulateExternalOperationWithError` returns `string` but should return `error` type for Go idiom

- [ ] **Unused return value**: `simulateExternalOperationWithError()` result is ignored in `deposit.go` after Invalid Account removal - clean up

## Optional Improvements

- [ ] **Merge `messages` package**: Could merge into `workflows` since signals/queries are tightly coupled to workflows

- [ ] **Consistent file naming**: `account_transfer_workflow_scenarios.go` is long - could simplify to `scenarios.go` within the `workflows` package
