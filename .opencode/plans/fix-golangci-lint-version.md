# Fix golangci-lint Go version mismatch in CI

**Problem**: `go.mod` declares `go 1.26.3` but the pre-built `golangci-lint` v1.64.8 binary was compiled with Go 1.24, causing the lint step to fail with: `can't load config: the Go language version (go1.24) used to build golangci-lint is lower than the targeted Go version (1.26.3)`

**Fix**: Option A — add `install-mode: goinstall` to the golangci-lint action step.

## Changes

### 1. `.github/workflows/ci.yml`
Add `install-mode: goinstall` to the Lint step so golangci-lint is compiled from source using our Go 1.26, avoiding the version mismatch.

```diff
       - name: Lint
         uses: golangci/golangci-lint-action@v6
         with:
           version: latest
+          install-mode: goinstall
```

### 2. `ci_test.go`
No changes needed — the existing `TestCIWorkflowHasLintCheck` checks for `"Vet"` or `"lint"` and the step is named `"Lint"`.

## Verification

1. Run `go test ./... -v` — all tests should pass (no test changes needed)
2. Run `go vet ./...` and `.github/check-imports.sh` — no regressions
3. Commit, push to `working`, open a PR

## Post-merge

The CI lint step will compile golangci-lint from source on each run, adding ~30s to first install but preventing future Go version mismatches.
