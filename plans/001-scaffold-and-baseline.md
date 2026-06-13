# Plan 001: Establish project scaffold, dependencies, and a green verification baseline

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md`.
>
> **Drift check (run first)**: This is a greenfield build with no git history.
> Run `ls` in the repo root. You should see only `PLAN.md` and `plans/`. If you
> also see `go.mod`, `main.go`, `puzzle/`, or `render/` already present, the
> repo has drifted from the assumption that this is a fresh build — treat that
> as a STOP condition and report what exists.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: dx
- **Planned at**: no git repo (greenfield), 2026-06-13

## Why this matters

Nothing can be tested until there is a buildable Go module with a working test
command. This plan creates the module, pins dependencies, places the embedded
font, and proves `go build` / `go vet` / `go test` all run green on a trivial
test. Every later plan uses these exact commands as its verification gate, so
this baseline must exist and pass first.

## Current state

The repo contains only `PLAN.md` (the design spec) and `plans/`. There is no
`go.mod`, no source files, and no `fonts/` directory. `PLAN.md` is the
authoritative design; read it once for context, but this plan supersedes its
library choices where they differ (notably: use `golang.org/x/image/font/opentype`,
**not** `github.com/golang/freetype`).

Target structure after the full plan set:

```
sudoprint/
├── go.mod
├── main.go
├── puzzle/
│   ├── generate.go
│   ├── generate_test.go
│   ├── solve.go
│   └── solve_test.go
├── render/
│   ├── image.go
│   ├── image_test.go
│   ├── pdf.go
│   └── pdf_test.go
└── fonts/
    └── JetBrainsMono-Regular.ttf
```

This plan creates only the module, dependency set, font asset, and a placeholder
package with one passing test.

## Commands you will need

| Purpose   | Command            | Expected on success      |
|-----------|--------------------|--------------------------|
| Init mod  | `go mod init sudoprint` | creates `go.mod`    |
| Tidy deps | `go mod tidy`      | exit 0, populates `go.sum` |
| Build     | `go build ./...`   | exit 0, no output        |
| Vet       | `go vet ./...`     | exit 0, no output        |
| Test      | `go test ./...`    | exit 0, `ok` per package |

The module path is `sudoprint`; internal imports are `sudoprint/puzzle` and
`sudoprint/render`.

## Scope

**In scope** (create these):
- `go.mod`
- `fonts/JetBrainsMono-Regular.ttf` (download — see Step 2)
- `puzzle/doc.go` (temporary placeholder package + one test)
- `puzzle/doc_test.go`

**Out of scope** (do NOT create yet — later plans own them):
- `puzzle/solve.go`, `puzzle/generate.go` (plans 002, 003)
- `render/*` (plans 004, 005)
- `main.go` (plan 006)

## Git workflow

The repo is not a git repository. Do NOT run `git init`, commit, or push unless
the operator explicitly asks. Just create files on disk.

## Steps

### Step 1: Initialize the module and pin dependencies

Run from the repo root:

```
go mod init sudoprint
```

Set the Go version line in `go.mod` to `go 1.22`.

Then add the three runtime dependencies (later plans import them; adding now
proves they resolve):

```
go get golang.org/x/image/font/opentype@latest
go get github.com/signintech/gopdf@latest
```

`golang.org/x/image` (the parent module of `font/opentype`, `font`,
`math/fixed`, `draw`) is pulled in by the first `go get`. Do **not** add
`github.com/golang/freetype` — it is frozen/unmaintained and this build uses
`x/image/font/opentype` instead.

**Verify**: `go mod tidy` → exit 0. Then confirm `go.mod` `require` block lists
`golang.org/x/image` and `github.com/signintech/gopdf`.

### Step 2: Obtain and place the embedded font

Create `fonts/`. Download **JetBrains Mono Regular** (OFL license, free to embed)
from the official release assets at
`https://github.com/JetBrains/JetBrainsMono/releases` and save the regular
weight as exactly:

```
fonts/JetBrainsMono-Regular.ttf
```

The file inside the release zip is typically at `fonts/ttf/JetBrainsMono-Regular.ttf`.

**Verify**: the file exists and is a real TTF (size is roughly 200 KB or larger,
not a 0-byte or HTML error page). On a Unix shell:
`test -s fonts/JetBrainsMono-Regular.ttf && echo OK` → prints `OK`.

If you cannot download the file (no network access in your environment), **STOP**
and report: "font download blocked — operator must place
`fonts/JetBrainsMono-Regular.ttf` manually." Do not substitute a different font.

### Step 3: Create a placeholder package with one passing test

So `go test ./...` has something to run before the real packages exist, create:

`puzzle/doc.go`:
```go
// Package puzzle generates sudoku grids and validates their uniqueness.
package puzzle
```

`puzzle/doc_test.go`:
```go
package puzzle

import "testing"

func TestScaffold(t *testing.T) {
	if 1+1 != 2 {
		t.Fatal("arithmetic is broken; environment is not sane")
	}
}
```

(Plan 002 replaces `doc.go` with real solver code and may delete this test once
real tests exist — that is expected.)

**Verify**: `go test ./...` → exit 0, output includes `ok  	sudoprint/puzzle`.

### Step 4: Confirm the full gate is green

Run all three gate commands in sequence.

**Verify**:
- `go build ./...` → exit 0, no output
- `go vet ./...` → exit 0, no output
- `go test ./...` → exit 0, `ok` for `sudoprint/puzzle`

## Test plan

- One trivial test (`TestScaffold`) exists solely to prove the test runner
  works. No real logic is tested in this plan.
- Verification: `go test ./...` → all pass.

## Done criteria

ALL must hold:

- [ ] `go.mod` exists with module `sudoprint` and `go 1.22`
- [ ] `go.mod` requires `golang.org/x/image` and `github.com/signintech/gopdf`; does NOT require `github.com/golang/freetype`
- [ ] `fonts/JetBrainsMono-Regular.ttf` exists and is non-empty
- [ ] `go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] `plans/README.md` status row for 001 updated to DONE

## STOP conditions

Stop and report back (do not improvise) if:

- Source files (`go.mod`, `main.go`, `puzzle/solve.go`, etc.) already exist
  before you start.
- The font cannot be downloaded in your environment (see Step 2).
- `go get` cannot resolve `golang.org/x/image` or `github.com/signintech/gopdf`
  (network/proxy issue) — report the exact error.

## Maintenance notes

- If the Go toolchain available is older than 1.22, lower the `go.mod` directive
  to match it; nothing in this build requires 1.22-specific features, but do not
  go below 1.18 (needed by `x/image/font/opentype` and generics-free code here).
- The font is embedded via `go:embed` in plan 004. Keep the path
  `fonts/JetBrainsMono-Regular.ttf` stable; changing it breaks the embed directive.
- Reviewer should confirm `github.com/golang/freetype` never appears in `go.mod`
  or any import — the legacy library was deliberately rejected.
