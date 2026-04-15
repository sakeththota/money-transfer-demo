# Code Convention TODOs

## Go Conventions

- [x] **Module name**: Rename `money-transfer-worker` to `money-transfer-demo` or a proper import path like `github.com/user/money-transfer` (reflects whole project, not just worker)

- [x] **Package `app`**: Renamed to `transfer` (name packages by what they provide, not generic containers)

- [x] **SCREAMING_CASE constants**: Change to Go-style CamelCase
  - `ADVANCED_VISIBILTY` → `AdvancedVisibility` (also fix typo - missing "I")
  - `NEEDS_APPROVAL` → `NeedsApproval`
  - `BUG` → `Bug`
  - `APPROVAL_TIME` → `ApprovalTime`
  - `API_DOWNTIME` → `APIDowntime`

- [x] **Shadowed builtin**: In `activities/withdraw.go`, `error` variable shadows builtin type. Renamed to use inline `err`

- [x] **Function signature**: `simulateExternalOperationWithError` returns `string` but should return `error` type for Go idiom

- [x] **Unused return value**: `simulateExternalOperationWithError()` now returns error and is properly handled in both `deposit.go` and `withdraw.go`

## Optional Improvements

- [x] **Merge `messages` package**: Merged into `workflows/signals.go` since signals/queries are tightly coupled to workflows

- [x] **Consistent file naming**: Renamed `account_transfer_workflow_scenarios.go` → `scenarios.go`
