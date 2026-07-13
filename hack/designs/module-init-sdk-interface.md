# Implement the engine SDK interface for `dagger module init dang`

- **Status:** Done
- **Date:** 2026-07-13
- **Affects:** `dang-sdk.dang` (the `DangSdk` root type), templates, e2e, sdk-sdk contract
- **Engine baseline:** `dagger` `v1.0.0-beta.6`

## Summary

Starting with the CLI-1.0 line (`v1.0.0-beta.6`), module creation moved out of
SDK-specific `dagger call` helpers and into a first-class CLI command:

```sh
dagger module init <sdk> <name> [--path=.]
```

The engine drives this through a well-known GraphQL contract. To be usable as an
SDK for `dagger module init`, the SDK **module** must expose two functions on its
root type:

- `initModule(ws: Workspace!, name: String!, path: String!, …): Changeset!`
- `targetRuntime: String!`

`dang-sdk` (this repo) is the module the engine resolves for the built-in `dang`
SDK, but it implements neither. It exposes `init(...)` (wrong name, and it writes
the now engine-owned config file) and no `targetRuntime`. As a result:

```sh
dagger --x-release=v1.0.0-beta.6 module init dang deno --path=.
# Error: unknown command "dang" for "dagger module init"
```

This document explains the new contract and proposes the minimal change to make
`dang-sdk` satisfy it.

## Problem

Running, in a fresh workspace under `~/Documents/github.com/dagger/deno`:

```sh
dagger --x-release=v1.0.0-beta.6 module init dang deno --path=.
```

fails with a cobra `unknown command "dang"` error. It never reaches the engine.

### Why it fails

`dagger module init` no longer takes `<sdk>` as a positional argument. The parent
command declares `Args: cobra.NoArgs`; the `<sdk>` slot is filled by
**dynamically registered subcommands**, one per SDK that is (a) installed in the
workspace `dagger.toml` as an `[modules.<x>.as-sdk]` entry, and (b) actually
exposes an `initModule` function.

- CLI parent + dynamic registration:
  `internal/cmd/dagger/module_init.go` (`moduleInitCmd`),
  `internal/cmd/dagger/sdk_init_dynamic.go`
  (`registerInstalledSDKInitCommands` → `configuredSDKs` → `inspectSDKInitFunction`,
  which looks for the function named `initModule`).
- Only when both conditions hold is a `dang <name>` subcommand added.

For the command above, neither condition holds:

1. `dang` is not installed as an as-sdk entry in the workspace, so no subcommand
   is produced (`configuredSDKs` skips every entry whose `AsSDK == nil`).
2. Even after `dagger sdk install dang` (which records
   `github.com/dagger/dang-sdk` as an as-sdk entry named `dang`), the subcommand
   still would not register, because the `dang-sdk` module does not expose
   `initModule` — `inspectSDKInitFunction` returns "not found" and the entry is
   skipped.

So the user sees `unknown command "dang"`, not an engine-side message.

### A second, latent failure

Even if `initModule` existed and the subcommand registered, a module created by
`dagger module init dang deno` would be **broken at call time**. When an SDK does
not expose `targetRuntime`, the engine records the SDK's own installed ref as the
new module's `[runtime] source`:

```go
// core/schema/workspace_module_init.go — resolveModuleRuntimeRef
if override, ok := s.lookupSDKTargetRuntime(ctx, sdkRef); ok && override != "" {
    return override, nil
}
return sdkRef, nil // sdkRef == "github.com/dagger/dang-sdk"
```

The new module would then declare `runtime = github.com/dagger/dang-sdk`, but
`dang-sdk` implements no runtime (`moduleRuntime`/`codegen`), so `dagger call`
against the new module would fail. The Dang runtime is the engine's built-in
`dang` interpreter, so `dang-sdk` must delegate to it via `targetRuntime → "dang"`.

## Background: the CLI-1.0 module-init contract

This is the interface `dang-sdk` must satisfy. All references are to
`github.com/dagger/dagger` at tag `v1.0.0-beta.6`.

### 1. CLI is a thin wrapper over `Workspace.moduleInit`

`dagger module init <sdk> <name>` marshals SDK-specific flags into a JSON blob
and issues:

```graphql
mutation ($name: String!, $sdk: String!, $path: String, $args: JSON) {
  currentWorkspace {
    moduleInit(name: $name, sdk: $sdk, path: $path, args: $args) { id }
  }
}
```

- CLI call: `internal/cmd/dagger/module_init.go` (`callModuleInit`).
- Schema field: `core/schema/workspace.go` (`dagql.Func("moduleInit", …)`);
  resolver `core/schema/workspace_module_init.go` (`moduleInit`).
- Field signature (`docs/docs-graphql/schema.graphqls`):

  ```graphql
  moduleInit(
    name: String!
    sdk: String = ""
    path: String = ""
    source: String = ""
    include: [String!] = []
    here: Boolean = false
    args: JSON
  ): Changeset!
  ```

The result is a `Changeset`, previewed/applied through the standard changeset
flow.

### 2. SDK name resolution requires an installed as-sdk entry

`moduleInit` resolves `sdk` via `installedSDKSource(cfg, args.SDK)`
(`core/schema/workspace_sdk.go`). This requires the workspace `dagger.toml` to
contain a module entry whose key is the SDK name, or whose `[…as-sdk] name`
equals it, with `AsSDK != nil`. Otherwise:

```
"<sdk>" is not installed as an SDK in this workspace; run `dagger sdk install <sdk>` first
```

The built-in name registry (`core/sdk/sdkmeta`, which lists `dang`) governs only
**runtime/SDK loading** (`core/sdk/loader.go`), *not* `moduleInit`. There is no
"install-free" path for `dagger module init`.

The engine maps the built-in `dang` name to this repo:

```go
// core/sdk/workspace_module.go — workspaceModuleForBuiltinSDK
case sdkDang:
    return WorkspaceModule{Name: "dang-sdk", Source: "github.com/dagger/dang-sdk"}, true
```

So `dagger sdk install dang` records `github.com/dagger/dang-sdk` as the as-sdk
module aliased to the name `dang`.

### 3. Capability detection is name-based, on the module's root type

The engine scans the SDK module's **main object** (the type named after the
module) for well-known function names
(`core/sdk/consts.go`):

```go
var sdkFunctions = []string{
    "withConfig", "codegen", "moduleRuntime", "moduleTypes",
    "requiredClientGenerationFiles", "generateClient",
    "initModule", "initClient", "targetRuntime",
}
```

Presence of a name enables the corresponding capability
(`core/sdk/module.go` — `AsModuleInitializer` gates on `funcs["initModule"]`,
`AsRuntimeTarget`/`TargetRuntime` on `funcs["targetRuntime"]`). Matching is
case-insensitive lower-camel (`initModule` == `InitModule` == `init_module`).
The umbrella interface (`core/sdk.go`) is a set of optional capability
conversions, so an SDK may implement only the parts it needs — this is the
"unbundled SDK interface" (`.changes/v0.18.10.md`, dagger#10525).

### 4. The `initModule` signature and the ownership split

Required shape (`core/sdk.go` — `ModuleInitializer`):

```graphql
initModule(
  ws: Workspace!    # matched by TYPE (core Workspace); parameter name is free
  name: String!     # matched by name; the engine injects <name>
  path: String!     # matched by name; the engine injects the resolved rel path
  # …SDK-specific args become CLI flags, delivered via args: JSON
): Changeset!
```

Argument binding (`core/sdk/module_init.go` — `initFunctionCallArgs`):

- the `Workspace`-typed argument is auto-injected (name irrelevant),
- `name` / `path` receive the standard values,
- any other declared arg is looked up by name in the `args` JSON,
- an `args` key with no matching parameter is a hard error.

**Ownership split** (`core/schema/workspace_module_init.go` —
`validateSDKInitChangesetOwnership`): the engine owns the config files and writes
them itself; the SDK's `initModule` Changeset owns everything else under the
module dir. The engine rejects an `initModule` result that writes any of:

- `dagger.toml` (workspace config),
- `<path>/dagger-module.toml` (module config),
- `<path>/dagger.json` (legacy module config),

with:

```
sdk "<sdk>" initModule must not modify engine-owned file(s): …
```

The engine merges its own config-only diff (updated `dagger.toml` + the new
`<path>/dagger-module.toml`) with the SDK's Changeset. It deliberately does **not**
run codegen at init.

### 5. Runtime vs. SDK split (`targetRuntime`)

`resolveModuleRuntimeRef` (`core/schema/workspace_module_init.go`) records the new
module's `[runtime] source`. If the SDK exposes `targetRuntime: String!` returning
a non-empty value, that value is recorded; otherwise the SDK's own install ref is.
`dang-sdk` is init-only and delegates execution to the engine's built-in `dang`
runtime, so it must return `"dang"`.

### 6. Config format

At `v1.0.0-beta.6` the native config format is TOML: `dagger.toml` (workspace) and
`dagger-module.toml` (module), with `dagger.json` still **read** as a legacy
module config (`core/workspace/detect.go`, `core/modules/config.go`). New modules
created by `moduleInit` are written by the engine as `dagger-module.toml`.

## Gaps in `dang-sdk` today

`DangSdk` (`dang-sdk.dang`) currently exposes:

- `init(ws, name, path, template, ignoreGenerated): Changeset!` — wrong name for
  the contract, and it writes `dagger.json` (line ~148,
  `.withNewFile(configPath, config)`), which the new engine forbids.
- `mod`, `modules`, `deps`, `engine`, `generate`, `templates` — management
  helpers, unaffected.

Missing:

1. `initModule` on `DangSdk` → CLI never registers the `dang` subcommand.
2. `targetRuntime` on `DangSdk` → new modules get a broken `[runtime] source`.

## Proposed change

Add two functions to the `DangSdk` root type — `targetRuntime` and `initModule`
— and remove the old `init`. `dagger module init dang` is now the single
module-creation entry point; the config-writing `init` (invoked via
`dagger call dang-sdk init`) is redundant and, being config-writing, is
incompatible with the CLI-1.0 ownership split anyway.

### `targetRuntime`

```dang
"""
Engine runtime recorded for modules created with this SDK.

The Dang runtime lives in the engine, so modules initialized through
`dagger module init dang` run on the built-in "dang" runtime rather than on
this management module.
"""
pub targetRuntime: String! {
  "dang"
}
```

### `initModule`

```dang
"""
Scaffold a new Dang module for `dagger module init dang <name>`.

The engine owns and writes the config files (dagger.toml, dagger-module.toml);
this returns only the starter source under `path`. `template` selects a
templates/<template> directory; empty uses the minimal template.
"""
pub initModule(ws: Workspace!, name: String!, path: String!, template: String! = ""): Changeset! {
  let modPath = if (path == "" or path == ".") { "." } else { path.trimSuffix("/") }
  let selectedTemplate = if (template == "") { "minimal" } else { template }

  if (currentModule.source.exists("templates/" + selectedTemplate) == false) {
    raise "unknown init template: " + template
  } else {
    polyfill.workspace(ws).fork
      .withDirectory(modPath, renderedTemplate(name, selectedTemplate))
      .changes
  }
}
```

Notes:

- The engine resolves `path` before calling us: an empty `--path` becomes
  `.dagger/modules/<name>`, and `--path=.` arrives as `.`. `initModule` therefore
  receives a concrete, workspace-relative path and does no `.dagger` find-up of
  its own (the engine handles default-path resolution).
- The returned Changeset adds only `<path>/main.dang` (via `renderedTemplate`,
  reused unchanged). It never touches `dagger.toml`, `dagger-module.toml`, or
  `dagger.json`, satisfying `validateSDKInitChangesetOwnership`.
- `template` is the only SDK-specific arg; the CLI surfaces it as
  `--template`. It is optional (has a default) so it does not become a required
  flag.
- The old `init`'s `ignoreGenerated` flag is not carried over: it configured
  `automaticGitignore` in the config that `init` wrote, which the engine now
  owns. Dang has no generated SDK files today (`generate` returns an empty
  changeset), so the knob is moot for this flow. If a config-level toggle is ever
  needed here, it belongs in the engine's config write, not in the SDK changeset.

### End-to-end flow after the change

```sh
# One-time, records [modules.…as-sdk] name = "dang" in dagger.toml
dagger sdk install dang

# Now the dang subcommand registers (initModule present) and works
dagger module init dang deno --path=.
```

The engine:

1. resolves `dang` → `github.com/dagger/dang-sdk`,
2. appends the authored-module path under `[…as-sdk].modules` and writes
   `dagger.toml`,
3. writes `deno`'s `dagger-module.toml` with `[runtime] source = "dang"`
   (from `targetRuntime`),
4. calls `dang-sdk.initModule(ws, "deno", ".", template: "")`, which adds
   `main.dang`,
5. merges the two changesets and prompts to apply.

## Backward compatibility

- **Breaking:** `dagger call dang-sdk init …` is removed. Module creation now
  goes through `dagger module init dang <name>`. The current `sdk-sdk` contract
  (which exercises `init` as a black-box CLI target) must be updated to the
  CLI-1.0 `initModule` contract in lockstep — see follow-ups.
- **`mod` / `deps` / `engine` / `generate` / `modules`** are untouched.
- Adding `initModule`/`targetRuntime` only *adds* capabilities; it cannot break
  the runtime path, which is the engine's built-in `dang` interpreter.

## Testing

- **e2e (`.dagger/modules/e2e`)**: `initModuleCheck` asserts the returned
  changeset adds `<path>/main.dang`, renders the expected root type, and adds
  **no** config file (see Verification for the added-files assertion).
  `targetRuntimeCheck` asserts `targetRuntime == "dang"`. Both are implemented and
  passing (below). The old `initCheck` is removed with `init`.
- **sdk-sdk contract (`github.com/dagger/sdk-sdk`)**: the shared contract in
  `sdk-sdk.dang` currently exercises `init`/`mod`/`deps`/`engine` as black-box CLI
  targets. It must swap `init` coverage for `initModule`/`targetRuntime` plus the
  ownership-split assertion so every official SDK helper is checked against the
  CLI-1.0 contract. Removing `init` here breaks that contract until it is updated.
  (Separate repo; tracked as follow-up.)

## Verification

Implemented and exercised against a `v1.0.0-beta.6` engine
(`dagger --x-release=v1.0.0-beta.6`):

- `dang-sdk.dang` gains `targetRuntime: String!` (returns `"dang"`) and
  `initModule(ws, name, path, template = ""): Changeset!`.
- e2e (`.dagger/modules/e2e`): added `targetRuntimeCheck` and `initModuleCheck`.
  `dagger --x-release=v1.0.0-beta.6 -m .dagger/modules/e2e check` → **11/11 pass**.
  `initModuleCheck` asserts the changeset adds only the starter `main.dang` (added
  *files*, ignoring created directory entries), renders the expected root type,
  and writes no config file — the ownership split.
- CLI registration (the original blocker): after
  `dagger sdk install <dang-sdk> --name dang`, the `dang` subcommand now appears
  under `dagger module init` (`dang  Initialize a new module with dang`), so the
  original `unknown command "dang"` is resolved. This is gated on `initModule`
  being present (`inspectSDKInitFunction`); with the old `init`-only module it
  does not register.

Not reproducible locally (engine limitation, unrelated to this repo): completing
`dagger module init dang <name>` against a **locally vendored** SDK fails at
`load sdk "<path>": local module dep source path must be relative to a parent
module`. The engine's `moduleInit` loads the SDK via `loadWorkspaceSDK` with a
`nil` parent (`core/schema/workspace_sdk_init.go`), which rejects local
workspace-relative sources; it happens at `resolveModuleRuntimeRef` →
`lookupSDKTargetRuntime` before `initModule` is ever called. The production
`dang` SDK resolves to the git ref `github.com/dagger/dang-sdk`, which loads
fine, so this only affects local vendoring during development.

## Out of scope / upstream follow-ups

- **Install UX.** `dagger module init dang` requires a prior `dagger sdk install
  dang`; otherwise the subcommand never registers and the user sees
  `unknown command "dang"` rather than a hint to install. Whether `module init`
  should auto-install or emit a better hint is an engine/CLI concern, not fixable
  in this repo. Worth raising upstream.
- **Local-path SDK sources.** `moduleInit` cannot load a locally-vendored SDK
  (`loadWorkspaceSDK` uses a `nil` parent, so a workspace-relative source is
  rejected — see Verification). Only affects offline/dev vendoring; the git ref
  works. Worth raising upstream.
- **TOML migration.** This repo's own `dagger.json` files and the e2e fixtures
  still use `dagger.json` (read as legacy). Migrating them to `dagger-module.toml`
  is a broader change and is not required for `module init` to work.
- **Polyfill nested-generate workaround.** `polyfill`'s `PolyfillGeneration` still
  shells out to a privileged nested Go helper "until the engine exposes a direct
  API usable from Dang." `initModule` does not depend on it (it only uses
  `fork` / `withDirectory` / `changes`), so removing that workaround is
  independent of this change.

## References (dagger/dagger @ v1.0.0-beta.6)

- `internal/cmd/dagger/module_init.go` — CLI parent + `callModuleInit`.
- `internal/cmd/dagger/sdk_init_dynamic.go` — dynamic subcommand registration,
  `inspectSDKInitFunction`, `configuredSDKs`.
- `core/schema/workspace.go` — `moduleInit` field registration.
- `core/schema/workspace_module_init.go` — resolver, `resolveModuleRuntimeRef`,
  `validateSDKInitChangesetOwnership`.
- `core/schema/workspace_sdk.go` — `installedSDKSource`.
- `core/sdk.go` — `SDK`, `ModuleInitializer`, `RuntimeTarget` interfaces.
- `core/sdk/consts.go` — well-known SDK function names.
- `core/sdk/module.go`, `core/sdk/module_init.go` — capability detection + arg
  binding for module-based SDKs.
- `core/sdk/workspace_module.go`, `core/sdk/loader.go`, `core/sdk/sdkmeta` —
  built-in `dang` → `github.com/dagger/dang-sdk` mapping and runtime loading.
- `core/workspace/detect.go`, `core/modules/config.go` — TOML config filenames
  and legacy `dagger.json` reads.
