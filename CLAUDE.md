# CLAUDE.md

This is Bill Graziano's fork of [`github.com/d5/tengo`](https://github.com/d5/tengo)
(v2), the embeddable Go scripting language. It carries a small set of
general-purpose patches needed for embedding Tengo as a rules/transform
engine, and is consumed as an ordinary Go module dependency.

## Branches and remotes

| Remote | URL |
|---|---|
| `origin` | `github.com/billgraziano/tengo` (this fork) |
| `upstream` | `github.com/d5/tengo` |

- **`patches`** — the working branch. All fork changes live here. Consumers
  pin a pseudo-version from this branch.
- **`master`** — tracks upstream `d5/tengo` master. It is currently **ahead**
  of the point `patches` was branched from (`c461a7f`): it has upstream
  commits (e.g. `freeze()` builtin, int==float fix, regex alternation fix)
  that are not yet in `patches`. Bringing them in means rebasing or merging
  `patches` onto `master` and resolving any import-path conflicts from the
  module rename.
- No tags. Consumers use a pseudo-version, e.g.
  `require github.com/billgraziano/tengo/v2 v2.0.0-20260929125720-d678482839e7`.

## Module path

The module is renamed to **`github.com/billgraziano/tengo/v2`** (commit
`d678482`). Every internal import (`parser`, `stdlib`, `require`, `token`,
etc.) was rewritten accordingly, so it's a drop-in, no-`replace` dependency.
When pulling upstream changes, any new file importing
`github.com/d5/tengo/v2/...` must be rewritten to the fork path.

## Fork changes (on `patches`, one commit each)

The patches are independent of one another. Each is meant to be a
general-purpose improvement that makes sense in *any* embedding — nothing
project-specific belongs in the language/VM (see "Withdrawn" below).

### 1. Bare top-level `return` halts the script — `154a350`

Upstream rejects `return` outside a function. The fork allows a **bare**
`return` at the top level, which stops the script immediately (guard
clauses without a wrapper function):

```tengo
if event.user == "" { return }
```

- `compiler.go` (`*parser.ReturnStmt` case): at top level, a bare `return`
  compiles to `OpReturn 0`; `return <expr>` is a compile error
  `return with value not allowed outside function` (a value has no defined
  destination — results are read back via globals with `compiled.Get`).
- `vm.go` (`OpReturn` case): when `v.framesIndex == 1` there's no caller
  frame to pop, so it returns out of the run loop, the same as `OpSuspend`
  does at the natural end of the script. Without this, removing the
  compiler check would panic the VM.
- `return` inside a function is unchanged; `return` inside a top-level
  `for`/`if` halts the whole script.
- Existing tests updated for the new error text (`2d484b8`:
  `compiler_test.go`, `vm_test.go`).

Caveat for embedders: a global declared *inside* the script that is
skipped by an early `return` stays `undefined` (`Map()` returns `nil`, JSON
`null`). Inject result globals from Go (`script.Add("out", map[string]any{})`)
so they're always defined.

### 2. Reflective fallback in `FromInterface` — `aea6809`

Upstream `FromInterface` only handles a fixed set of Go types, so
`script.Add("x", []string{...})` fails with `cannot convert to object`.
The fork adds `fromReflectValue` (`tengo.go`) as the final case:

- Pointers/interfaces → deref (nil → `UndefinedValue`).
- All int/uint kinds → `int64`; float32/64 → `float64`; plus bool/string.
- Slices/arrays → `*Array`; maps → `*Map` (non-string keys via `fmt.Sprint`).
- Structs → `*Map` keyed like `encoding/json`: `json` tag name, else Go
  field name; `json:"-"` and unexported fields skipped. (Other tag options
  such as `omitempty` are ignored.)
- Every element recurses through `FromInterface`, so `MaxStringLen` /
  `MaxBytesLen` and existing `Object` passthrough still apply at any depth.
- Unsupported kinds (chan, func, complex, …) still return
  `cannot convert to object: %T`. Cyclic values recurse forever, same as
  `encoding/json`.

### 3. `ToGoValuer` hook in `ToInterface` — `54aaaf4`

Upstream `ToInterface` returns custom `Object` types unconverted. The fork
adds an opt-in interface (`tengo.go`):

```go
type ToGoValuer interface {
    ToGoValue() interface{}
}
```

checked **before** the generic `case Object` fallback (type-switch order
matters). A custom object implementing it round-trips to a plain Go value
through `Variable.Map()`/`Array()`/`ToInterface`, at any nesting depth.
Objects that don't implement it behave exactly as upstream.

### Withdrawn: `drop <target>` statement

A `drop out` keyword (reset a global to `{}` and halt from any call depth,
via a new `OpHalt` opcode) was built and then removed. It baked a
project-specific convention into the grammar, which conflicts with keeping
this fork generic. Don't reintroduce language features that only make sense
for one embedding; do that in the host (Go) code instead.

## Tests

- The fork's own test suite: `go test ./...` (upstream tests, with the
  `return` error-text updates above).
- **The patch-specific tests are not in this repo.** They live in the
  proof-of-concept at `C:\dev\poc\tengopoc`:
  `fork_test.go` (patch 1), `fork_patch2_test.go`, `fork_patch3_test.go`.
  That repo pins this fork by pseudo-version, so to test a change here,
  temporarily add `replace github.com/billgraziano/tengo/v2 => C:\dev\github.com\billgraziano\tengo`
  in its `go.mod` (and remove it afterward), or port the tests into this
  repo.
- Full design notes and rationale: `C:\dev\poc\tengopoc\tengo-fork.md`;
  variable-naming conventions for scripts: `tengo-variables.md` there.

## Consumers

- `C:\dev\poc\tengopoc` — proof of concept; its `rules` package (a
  `Program`/`Script` wrapper that injects `event`/`out`) is being copied
  into xelogstash.
- `C:\dev\github.com\xelogstash` — target application (`scripting` branch).
- Planned: one more future program.

After pushing to `patches`, update consumers with
`go get github.com/billgraziano/tengo/v2@patches`.

## Conventions

- Keep patches small, one commit per logical change, with a clear message —
  they should stay individually upstreamable to `d5/tengo`.
- Don't edit `README.md`/upstream docs for fork-only behavior; document it
  here.
- Match upstream style (the codebase uses `interface{}` in older code and
  `any` in newer code; follow the surrounding file).
