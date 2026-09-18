# Collections

Collection support requires [dagger/dagger#14221](https://github.com/dagger/dagger/pull/14221).

This SDK uses the engine's native Dang runtime. It does not generate a second
collection implementation. The runtime reads `@collection`, `@keys`, `@get`,
and `@delta`, and preserves the internal base when a function returns a changed
copy of its receiver. A new collection starts a new base.

The fixture in `.dagger/modules/e2e/fixtures/collections` checks the projected
API, subsets, arbitrary key changes, and a new collection. The check first runs
SDK scope generation to ensure that generation preserves the declarations.

```sh
dagger -m .dagger/modules/e2e check collections:run
```

The test builds the Collections engine commit named by `CollectionsTest.engineRef`.
