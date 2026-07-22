# dang-sdk

A Dagger module for managing Dagger modules that use the built-in Dang SDK.

It powers `dagger module init dang` and exposes the remaining
module-management operations (generate, dependencies, engine version) through
`dagger call`.

Backed by [`github.com/dagger/sdk-sdk/polyfill`](https://github.com/dagger/sdk-sdk/tree/main/polyfill).

## Install

From your workspace root:

```sh
dagger install github.com/dagger/dang-sdk
```

After install, the module is available in `dagger call` as `dang-sdk`.

Calls that return a `Changeset` will print the diff and prompt you to confirm
before writing anything to your workspace.

## Create A New Module

Install this module as the `dang` SDK, then create a module with the CLI:

```sh
dagger sdk install dang
dagger module init dang my-module
```

By default the module is created under `.dagger/modules/<name>/`. Pick a
different location with `--path`:

```sh
dagger module init dang my-module --path some/dir/my-module
```

`module init` seeds template files. Run `mod ... generate` to refresh generated
SDK files when generation produces any.

## Generate SDK Files

For a single module:

```sh
dagger call dang-sdk mod --path my-module generate
```

For every Dang SDK module visible from your current directory — the module
you're in and the projects beneath it — skipping any with a
`.dagger-dang-sdk-skip-generate` marker at or above the module root:

```sh
dagger call dang-sdk generate-all
```

## Manage Dependencies

List:

```sh
dagger call dang-sdk mod --path my-module deps list
```

Add:

```sh
dagger call dang-sdk mod --path my-module \
    deps add --source github.com/some/module
```

Add with a custom local name:

```sh
dagger call dang-sdk mod --path my-module \
    deps add --source github.com/some/module --name alias
```

Remove by name or source:

```sh
dagger call dang-sdk mod --path my-module deps remove --name alias
```

Update one remote dependency, or all of them:

```sh
dagger call dang-sdk mod --path my-module deps update
dagger call dang-sdk mod --path my-module deps update --name some-dep
```

## Manage The Required Engine Version

```sh
# Read the version pinned in dagger.json
dagger call dang-sdk mod --path my-module engine required

# Pin to a specific version
dagger call dang-sdk mod --path my-module engine require --version 0.20.8

# Pin to the engine version you're currently running
dagger call dang-sdk mod --path my-module engine require-current

# Pin to "latest"
dagger call dang-sdk mod --path my-module engine require-latest
```

## Discover Modules In A Workspace

Discovery is anchored at your current directory, not the workspace root: the
nearest enclosing module plus every Dang module beneath you. Modules configured
by the legacy `dagger.json` and the CLI 1.0 `dagger-module.toml` are both found.
Paths print relative to where you invoked the command.

```sh
dagger call dang-sdk modules path
```

See [`dang-sdk.dang`](./dang-sdk.dang) for the full type surface.

## Skipping Generation

To exclude a directory tree from `generate-all`, drop an empty
`.dagger-dang-sdk-skip-generate` file at or above the module root. Useful for
fixtures, vendored modules, or anything you do not want regenerated in bulk.

```sh
touch some/fixture/.dagger-dang-sdk-skip-generate
```

## Test

Run shared SDK helper contract checks from this repository with:

```sh
dagger -m github.com/dagger/sdk-sdk check
```

Run this repo's e2e module with:

```sh
dagger -m .dagger/modules/e2e check
```
