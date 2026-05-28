# dang-sdk

A Dagger module for managing Dagger modules that use the built-in Dang SDK.

The Dagger CLI ships without built-in module-management commands like `init`
or `develop`. Those operations live in SDK-specific modules like this one,
called through `dagger call`.

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

Create a Dang SDK module under the nearest `.dagger/modules/<name>/`:

```sh
dagger call dang-sdk init --name my-module
```

Pick a different location:

```sh
dagger call dang-sdk init --name my-module --path some/dir/my-module
```

`init` only seeds template files. Run `mod ... generate` to refresh generated
SDK files when generation produces any.

## Generate SDK Files

For a single module:

```sh
dagger call dang-sdk mod --path my-module generate
```

For every Dang SDK module in the workspace, skipping any with a
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
