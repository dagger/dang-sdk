# Collections

Use `@collection` to expose objects through a dynamic set of keys.
This requires an engine with [dagger/dagger#14221](https://github.com/dagger/dagger/pull/14221).
The engine contains the Dang runtime and the collection implementation.
Installing this SDK alone does not enable collections on an older engine.

Create a module:

```sh
dagger module init dang --name collections
```

Replace its `main.dang` with:

```dang
type Collections {
  items: Items! { Items(names: ["a", "b", "c"]) }
}

type Items @collection {
  pub names: [String!]! @keys
  pub selection: CollectionDelta @delta
  item(name: String!): Item! @get { Item(name: name) }
  added: [String!]! { selection!.addedKeys }
  removed: [String!]! { selection!.removedKeys }
  change(names: [String!]!): Items! {
    self.names = names
    self
  }
  fresh: Items! { Items(names: ["z"]) }
}

type Item { pub name: String! }
```

`@keys` selects the stored keys field. `@get` selects the function that returns
one item. You can omit these two directives if you name the members `keys` and
`get`.

The engine exposes `keys`, `get`, `list`, and `subset` to callers. It exposes
the other collection functions under `batch`. Each batch function runs once
for the selected collection.

Run this query:

```sh
dagger api query -m .dagger/modules/collections <<'GRAPHQL'
{
  items {
    subset(keys: ["b"]) {
      keys
      list { name }
      batch { added removed }
    }
  }
}
GRAPHQL
```

The result has key `b`, one item named `b`, no added keys, and removed keys
`a` and `c`.

The optional `@delta` field receives the changes from the original collection
before a function runs. The engine compares current keys with original keys.
This includes changes made by `subset` and by module functions.

For example, after `subset(keys: ["b"])`, calling
`batch.change(names: ["b", "d"])` keeps the original keys as the base.
Its next batch call sees added key `d` and removed keys `a` and `c`.
Calling `batch.fresh` constructs a new collection with no changes from its base.

The [collection check](../.dagger/modules/e2e/collections.dang) runs SDK generation
and checks these cases with the [example module](../.dagger/modules/e2e/fixtures/collections/main.dang).
