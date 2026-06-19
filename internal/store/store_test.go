package store_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"lucy/internal/store"
)

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// testStore connects to MONGODB_TEST_URI and returns a Store backed by a
// unique test database, dropped automatically on cleanup.
func testStore(t *testing.T) *store.Store {
	t.Helper()
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("MONGODB_TEST_URI not set — skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := store.Connect(ctx, uri, "lucy_test_"+t.Name())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() {
		dctx, dc := context.WithTimeout(context.Background(), 5*time.Second)
		defer dc()
		st.DropDatabase(dctx)
		st.Disconnect(dctx)
	})

	if err := st.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes: %v", err)
	}
	return st
}

func TestListCollectionsEmpty(t *testing.T) {
	st := testStore(t)
	cols, err := st.ListCollections(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 0 {
		t.Errorf("want 0 collections, got %d", len(cols))
	}
}

func TestEnsureCollectionIdempotent(t *testing.T) {
	st := testStore(t)

	id1, err := st.EnsureCollection(testCtx(t), "my_col")
	if err != nil {
		t.Fatal(err)
	}
	if id1 == (bson.ObjectID{}) {
		t.Fatal("got zero id")
	}

	id2, err := st.EnsureCollection(testCtx(t), "my_col")
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Errorf("second call returned different id: %v vs %v", id1, id2)
	}

	cols, err := st.ListCollections(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 1 {
		t.Errorf("want 1 collection, got %d", len(cols))
	}
	if cols[0].Name != "my_col" {
		t.Errorf("want name my_col, got %s", cols[0].Name)
	}
}

func TestGetSchemaNoEntry(t *testing.T) {
	st := testStore(t)
	id, err := st.EnsureCollection(testCtx(t), "empty_col")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := st.GetSchema(testCtx(t), id)
	if err != nil {
		t.Fatal(err)
	}
	if raw != nil {
		t.Errorf("expected nil schema, got %s", raw)
	}
}

func TestUpsertSchemaRoundTrip(t *testing.T) {
	st := testStore(t)
	id, err := st.EnsureCollection(testCtx(t), "schema_col")
	if err != nil {
		t.Fatal(err)
	}

	schema := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"score":{"type":"integer"}},"required":["name"]}`)
	if err := st.UpsertSchema(testCtx(t), id, schema); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetSchema(testCtx(t), id)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(schema) {
		t.Errorf("schema mismatch\nwant: %s\ngot:  %s", schema, got)
	}

	schema2 := json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}}}`)
	if err := st.UpsertSchema(testCtx(t), id, schema2); err != nil {
		t.Fatal(err)
	}
	got2, err := st.GetSchema(testCtx(t), id)
	if err != nil {
		t.Fatal(err)
	}
	if string(got2) != string(schema2) {
		t.Errorf("after overwrite, want %s, got %s", schema2, got2)
	}
}

func TestInsertItems(t *testing.T) {
	st := testStore(t)

	items := json.RawMessage(`[{"question":"What is PVC?","answer":"PersistentVolumeClaim"},{"question":"What is PV?","answer":"PersistentVolume"}]`)
	n, err := st.InsertItems(testCtx(t), "cka_exam", items, "pvc")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("want 2 inserted, got %d", n)
	}
}

func TestInsertItemsEmptyArray(t *testing.T) {
	st := testStore(t)
	n, err := st.InsertItems(testCtx(t), "col", json.RawMessage(`[]`), "")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("want 0, got %d", n)
	}
}
