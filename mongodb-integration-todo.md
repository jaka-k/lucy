# MongoDB integration — todo (next session)

Add MongoDB persistence to Lucy. Generated list items are written as documents
into a user-chosen **data collection**; each collection's schema is remembered
in a **schema registry** so picking a collection preloads its schema into the
visual builder. Mongo is **required**.

Workflow reminder: phased, commit after each phase, this file is gitignored.

---

## Decisions captured (from grilling)

- **Capabilities:** write items to a collection + reusable per-collection schemas (presets). No history browser / no run log beyond a tag.
- **Connection:** `MONGODB_URI` + `MONGODB_DB` in `.env` (**required** — app fails fast if missing/unreachable). Target collection chosen in the UI.
- **Doc shape:** one document **per item**; the only run metadata is a user-supplied **tag** (e.g. collection `cka_exam`, tag `pvc`) stamped on each committed doc.
- **Registry model:** `collections` registry + `schemas` registry, linked by FK (`schemas.collection_id -> collections._id`).
- **Schema preload:** selecting an existing collection **rebuilds the nested visual-builder tree** from its stored schema (reverse of `serializeBuilder` in app.js).
- **Write timing:** an **`auto-commit`** checkbox.
  - checked → every Generate inserts all items into the selected collection.
  - unchecked → after generation, a **modal** lists items with per-item checkboxes (default checked) + a **Commit** button; only checked items are inserted, unchecked are discarded.

---

## Proposed data model

`collections` registry:
```
{ _id: ObjectId, name: "cka_exam", created_at: ISODate }   // unique index on name
```
`schemas` registry:
```
{ _id: ObjectId, collection_id: ObjectId (FK), schema: <item JSON Schema>, updated_at: ISODate }  // unique index on collection_id
```
Data collection (e.g. `cka_exam`), one doc per generated item:
```
{ _id: ObjectId, ...item fields per schema..., tag: "pvc" }
```
Note: store the **item (object) schema** in the registry (pre-array-wrap), so it round-trips back into the builder; `schema.Build` still wraps it into an array at generate time.

---

## Dependencies / config

- [ ] `go get go.mongodb.org/mongo-driver/v2/mongo`  *(confirm v2 vs v1 at impl)*
- [ ] `config`: add required `MONGODB_URI`, `MONGODB_DB`; error if absent
- [ ] `.env` gains `MONGODB_URI`, `MONGODB_DB` (README + example)

---

## Phases

- [ ] **Phase 1 — Connection (required) + config**
  - [ ] add driver dep
  - [ ] config: require MONGODB_URI + MONGODB_DB
  - [ ] `internal/mongo` (or `internal/store`): `Connect(ctx)` with **ping**, `Disconnect` on shutdown
  - [ ] wire into `main` (fail fast on ping error) and pass store into `server.New`
  - commit: `feat(mongo): required connection and config`

- [ ] **Phase 2 — Registries**
  - [ ] `collections` + `schemas` registries with unique indexes (name; collection_id)
  - [ ] store ops: `ListCollections`, `EnsureCollection(name) -> id`, `GetSchema(collectionID)`, `UpsertSchema(collectionID, schema)`
  - commit: `feat(mongo): collections and schemas registries`

- [ ] **Phase 3 — UI: collection picker, tag, auto-commit**
  - [ ] collection `<select>` populated from registry + a "new collection" name input
  - [ ] `tag` input; `auto-commit` checkbox (decide default)
  - [ ] `GET /collections` (list) and `GET /collections/{id}/schema` endpoints
  - commit: `feat(web): collection picker, tag, auto-commit`

- [ ] **Phase 4 — Schema → builder round-trip**
  - [ ] on generate, upsert collection + store item schema
  - [ ] app.js: `deserializeBuilder(schema)` rebuilds the nested field tree (inverse of `serializeNode`/`serializeList`): object -> children, array -> item type, array-of-object -> nested children, required/description restored
  - [ ] on collection select -> fetch schema -> rebuild builder
  - commit: `feat(web): rebuild visual builder from stored schema`

- [ ] **Phase 5 — Generate + write (auto-commit path)**
  - [ ] generate flow: Gemini -> items; if auto-commit -> `InsertMany` (each doc + tag)
  - [ ] result panel: inserted count + collection + tag, alongside existing formatted preview/download
  - commit: `feat: write generated items to the selected collection`

- [ ] **Phase 6 — Preview modal + selective commit**
  - [ ] auto-commit off -> return items to client; modal with per-item checkboxes (default checked) + Commit
  - [ ] `POST /commit` inserts only selected items (+ collection + tag); discard unchecked
  - [ ] decide commit data path (see open decisions)
  - commit: `feat(web): preview modal with selective commit`

- [ ] **Phase 7 — Tests, README, manual e2e**
  - [ ] store tests against a test Mongo (see open decisions)
  - [ ] README: Mongo required, env vars, full flow
  - [ ] manual e2e: new collection -> generate -> commit; reselect collection -> schema rebuilds; tag filter sanity check
  - commit: `docs/test: mongo integration`

---

## Open decisions to confirm at start of next session

1. **Test/dev Mongo:** local `docker run mongo:7`, Atlas, or `testcontainers-go`? (Required runtime means dev needs an instance.)
2. **Driver version:** `mongo-driver/v2` (recommended) vs v1.
3. **`auto-commit` default:** checked (fast) or unchecked (safe preview)?
4. **Commit data path (Phase 6):** client re-submits selected items as JSON (stateless, simplest) vs server caches the last batch by id and client sends indices. Recommend re-submit for a local single-user tool.
5. **Per-doc metadata:** strictly `fields + tag`, or also stamp `created_at` (+ optional `schema_id` FK on each item)? You said "only the tag" — confirm we don't add created_at.
6. **New-collection naming:** validation (allowed chars / Mongo name rules); behaviour if the collection already exists with a *different* schema (overwrite registry schema? warn? version?).
7. **Schema versioning:** overwrite on change vs keep prior versions (currently assuming overwrite; no history).
8. **Format selector + Mongo:** keep JSON/YAML/XML/CSV preview+download in addition to writing to Mongo? (Assumed yes.)
9. **Tag cardinality:** single free-text tag per generation (assumed) vs multiple tags.
10. **BSON field order:** preserve via `bson.D` (matches schema order) vs `bson.M`. Minor.
11. **Out of scope confirm:** no history browser / no reading-back data-collection items in the UI for v1.
