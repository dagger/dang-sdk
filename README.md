# dang-sdk

A Dagger module for managing Dagger modules that use the built-in Dang SDK.

It implements the SDK provider interface used by `dagger module init`,
`dagger module client`, and `dagger generate`.

It uses the engine's native `Workspace` and `ModuleSource` APIs directly.

## Install

From your workspace root:

```sh
dagger module install github.com/dagger/dang-sdk
```

The install registers the module as the `dang` SDK.

## Create A New Module

Create a module with the CLI:

```sh
dagger module init dang --name my-module
```

By default the module is created under `.dagger/modules/<name>/`. Pick a
different location with `--path`:

```sh
dagger module init dang --name my-module --path some/dir/my-module
```

`module init` writes `dagger-module.toml` and the minimal `main.dang` template.
The engine records the scope in `dagger.toml`.

The template is an SDK scope setting. The only current value is `minimal`:

```sh
dagger module init dang --name my-module --template minimal
```

## Generate Scope Files

The engine calls `generateScope` for each recorded Dang scope:

```sh
dagger generate
```

The SDK creates files only when a module has no `dagger-module.toml`. It keeps
existing module source during later generation.

## Manage Module Clients

The built-in Dang runtime has no generated client files. This SDK stores the
complete client set as module dependencies in `dagger-module.toml`.

Add, list, update, or remove a module client with the standard commands:

```sh
dagger module client add <module>
dagger module client list
dagger module client update
dagger module client rm <module>
```

## Direct Module Helpers

The module keeps direct helpers for legacy `dagger.json` modules. Calls that
return a `Changeset` show the diff and ask for confirmation.

List dependencies:

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

# Pin to the latest release the engine knows about
dagger call dang-sdk mod --path my-module engine require-latest
```

See [`dang-sdk.dang`](./dang-sdk.dang) for the full type surface.

## Skipping Generation

The direct `mod generate` helper skips a module when it finds a
`.dagger-dang-sdk-skip-generate` file at or above the module root.

```sh
touch some/fixture/.dagger-dang-sdk-skip-generate
```

## Test

Run the e2e module with:

```sh
dagger -m .dagger/modules/e2e check
```
