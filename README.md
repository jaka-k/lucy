# Lucy

A small Go web service that turns a **prompt** + a **JSON Schema** into an
LLM-generated **list**, then writes the items to MongoDB and serves them in the
format you pick: JSON, YAML, XML, or CSV.

Gemini always returns JSON (constrained by the schema via native structured
output); the other formats are produced locally with standard data libraries.

## Requirements

- Go 1.26+
- A Gemini API key
- MongoDB 7 (required — the app fails fast if unreachable)

## Quick start

```sh
# 1. Start MongoDB
docker compose up -d

# 2. Copy and fill in the env file
cp .env.example .env   # edit GEMINI_API_KEY

# 3. Run
go run ./cmd/lucy
```

Then open http://localhost:8077.

### Hot reload (dev)

A [`air`](https://github.com/air-verse/air) config is included; it rebuilds on
changes to Go sources, templates, and static assets (the last two are embedded
via `go:embed`, so rebuild is required).

```sh
go install github.com/air-verse/air@latest   # one-time
air                                          # in the project root
```

### Configuration via `.env`

Drop a `.env` file in the working directory (it's gitignored). Real environment
variables take precedence over values in `.env`.

```sh
# .env
GEMINI_API_KEY="your-key-here"
PORT=8077
MONGODB_URI=mongodb://localhost:27017
MONGODB_DB=lucy
```

### Environment variables

| Variable                            | Default      | Purpose                          |
| ----------------------------------- | ------------ | -------------------------------- |
| `GEMINI_API_KEY` / `GOOGLE_API_KEY` | — (required) | Gemini API key                   |
| `PORT`                              | `8080`       | HTTP listen port                 |
| `MONGODB_URI`                       | — (required) | MongoDB connection string        |
| `MONGODB_DB`                        | — (required) | Database name (e.g. `lucy`)      |

The model is **not** configured here. At startup Lucy queries Gemini's
ListModels endpoint and offers the `generateContent`-capable models in a UI
dropdown (falling back to a small static list if the call fails).

## Using it

1. Write a **prompt**, e.g. _"Create questions for the CKA exam segment on
   persistent volumes."_
2. Define the **output schema**, either:
   - **Visual builder** — add fields (name, type, optional array item type,
     description, required). Describes a single list item.
   - **Raw JSON Schema** — paste a schema directly.
3. Pick a **MongoDB collection** (existing or type a new name), an optional
   **tag**, and whether to **auto-commit**.
4. Pick a **format**, a **count**, and optionally a **model**.
5. **Generate**.
   - **Auto-commit on** → items are inserted immediately; the result panel
     shows the inserted count, collection, and tag.
   - **Auto-commit off** → a **Commit** modal appears with per-item checkboxes
     (all checked by default); only the checked items are inserted.
6. Download the result with one click (JSON / YAML / XML / CSV).

### Schema round-trip

Selecting an existing collection from the dropdown fetches its stored schema
and **rebuilds the visual builder** so you can tweak and re-generate without
re-entering the field definitions.

### Schema auto-detection

- A top-level `"type": "array"` schema is sent to Gemini as-is.
- Any other schema is treated as a single item and wrapped in an array, so the
  model returns a list of those items.
- `count` (when > 0) constrains the list to exactly that many items.

## Data model

`collections` registry — one doc per named collection:
```json
{ "_id": ObjectId, "name": "cka_exam", "created_at": ISODate }
```

`schemas` registry — one item schema per collection (overwritten on each generate):
```json
{ "_id": ObjectId, "collection_id": ObjectId, "schema": {…}, "updated_at": ISODate }
```

Data collection (e.g. `cka_exam`) — one doc per generated item:
```json
{ "_id": ObjectId, …item fields…, "tag": "pvc", "created_at": ISODate }
```

## Testing

Store integration tests require a running MongoDB:

```sh
docker compose up -d
MONGODB_TEST_URI=mongodb://localhost:27017 go test ./internal/store/ -v
```

Without `MONGODB_TEST_URI` the tests skip gracefully.

All other tests run without any infrastructure:

```sh
go test ./...
```

## Layout

```
cmd/lucy            entrypoint
internal/config     env-based configuration
internal/gemini     google.golang.org/genai wrapper (structured output)
internal/schema     JSON Schema parse, auto-detect, convert to genai.Schema
internal/convert    JSON -> json / yaml / xml / csv
internal/store      MongoDB connection, registries, item writes
internal/server     HTTP router + handlers
web                 embedded htmx templates and static assets
docker-compose.yml  local MongoDB (mongo:7)
```
