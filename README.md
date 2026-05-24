# Lucy

A small Go web service that turns a **prompt** + a **JSON Schema** into an
LLM-generated **list**, then serves it in the format you pick: JSON, YAML, XML,
or CSV.

Gemini always returns JSON (constrained by the schema via native structured
output); the other formats are produced locally with standard data libraries.

## Requirements

- Go 1.26+
- A Gemini API key

## Run

```sh
export GEMINI_API_KEY=your-key-here   # or GOOGLE_API_KEY
go run ./cmd/lucy
```

Then open http://localhost:8080.

### Configuration via `.env`

Instead of exporting variables, you can drop a `.env` file in the working
directory (it's gitignored). Real environment variables take precedence over
values in `.env`.

```sh
# .env
GEMINI_API_KEY="your-key-here"
PORT=8080
```

### Environment

| Variable                            | Default       | Purpose          |
| ----------------------------------- | ------------- | ---------------- |
| `GEMINI_API_KEY` / `GOOGLE_API_KEY` | — (required)  | Gemini API key   |
| `PORT`                              | `8080`        | HTTP listen port |

The model is **not** configured here. At startup Lucy queries Gemini's
ListModels endpoint and offers the `generateContent`-capable models in a UI
dropdown (falling back to a small static list if the call fails).

## Using it

1. Write a **prompt**, e.g. _"Create questions for the CKA exam segment on
   persistent volumes."_
2. Define the **output schema**, either:
   - **Visual builder** — add fields (name, type, optional array item type,
     description, required). This describes a single list item.
   - **Raw JSON Schema** — paste a schema directly.
3. Pick a **format**, a **count**, and optionally a **model**.
4. **Generate**. Download the result with one click.

### Schema auto-detection

- A top-level `"type": "array"` schema is sent to Gemini as-is.
- Any other schema is treated as a single item and wrapped in an array, so the
  model returns a list of those items.
- `count` (when > 0) constrains the list to exactly that many items.

## Layout

```
cmd/lucy        entrypoint
internal/config env-based configuration
internal/gemini google.golang.org/genai wrapper (structured output)
internal/schema JSON Schema parse, auto-detect, convert to genai.Schema
internal/convert JSON -> json / yaml / xml / csv
internal/builder visual-builder fields -> JSON Schema
internal/server HTTP router + handlers
web             embedded htmx templates and static assets
```
