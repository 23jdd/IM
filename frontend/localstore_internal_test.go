package main

import "testing"

func TestLocalStoreSearchMessages(t *testing.T) {
	t.Setenv("IM_DATA_DIR", t.TempDir())

	store := NewLocalStore()
	defer closeTestLocalStore(store)
	if err := store.Init("u1"); err != nil {
		t.Fatalf("init local store: %v", err)
	}
	if err := store.SaveMessage("peer", "m1", "u1", "hello first", true, "sent", 1000); err != nil {
		t.Fatalf("save m1: %v", err)
	}
	if err := store.SaveMessage("peer", "m2", "u2", "other text", false, "recv", 2000); err != nil {
		t.Fatalf("save m2: %v", err)
	}
	if err := store.SaveMessage("peer", "m3", "u2", "hello again", false, "recv", 3000); err != nil {
		t.Fatalf("save m3: %v", err)
	}

	got, err := store.SearchMessages("peer", "hello", 10)
	if err != nil {
		t.Fatalf("search messages: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 search results, got %d", len(got))
	}
	if got[0].MsgId != "m3" || got[1].MsgId != "m1" {
		t.Fatalf("expected newest first m3,m1, got %#v", got)
	}
}

func TestLocalStoreClearMessagesOnlyPeer(t *testing.T) {
	t.Setenv("IM_DATA_DIR", t.TempDir())

	store := NewLocalStore()
	defer closeTestLocalStore(store)
	if err := store.Init("u1"); err != nil {
		t.Fatalf("init local store: %v", err)
	}
	if err := store.SaveMessage("peer-a", "m1", "u1", "hello", true, "sent", 1000); err != nil {
		t.Fatalf("save peer-a: %v", err)
	}
	if err := store.SaveMessage("peer-b", "m2", "u2", "keep", false, "recv", 2000); err != nil {
		t.Fatalf("save peer-b: %v", err)
	}
	if err := store.ClearMessages("peer-a"); err != nil {
		t.Fatalf("clear peer-a: %v", err)
	}

	cleared, err := store.LoadMessages("peer-a", 10)
	if err != nil {
		t.Fatalf("load cleared peer: %v", err)
	}
	if len(cleared) != 0 {
		t.Fatalf("expected peer-a cleared, got %#v", cleared)
	}

	kept, err := store.LoadMessages("peer-b", 10)
	if err != nil {
		t.Fatalf("load kept peer: %v", err)
	}
	if len(kept) != 1 || kept[0].MsgId != "m2" {
		t.Fatalf("expected peer-b to stay, got %#v", kept)
	}
}
func closeTestLocalStore(store *LocalStore) {
	if store.db != nil {
		_ = store.db.Close()
	}
	if store.sessionDB != nil {
		_ = store.sessionDB.Close()
	}
}
