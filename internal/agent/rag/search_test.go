package rag

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalSearcherRanksAndBoundsDocuments(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(docs, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "release.md"), []byte("# Release Guide\n生产发布前执行 dry-run 和分批验证。"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "unrelated.txt"), []byte("database backup notes"), 0o600); err != nil {
		t.Fatal(err)
	}
	searcher := LocalSearcher{Root: root, Paths: []string{"docs"}, MaxDocuments: 1, MaxChars: 20}
	results, err := searcher.Search(context.Background(), "生产发布 dry-run")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0].Path != "docs/release.md" {
		t.Fatalf("results = %+v", results)
	}
	if len(results[0].Excerpt) > 20 {
		t.Fatalf("excerpt length = %d", len(results[0].Excerpt))
	}
	if PromptContext(results) == "" {
		t.Fatal("PromptContext() is empty")
	}
}

func TestLocalSearcherListsMarkdownAndTextOnly(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.md", "b.txt", "ignored.json"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("# title"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	results, err := (LocalSearcher{Root: root, Paths: []string{"."}}).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("documents = %+v", results)
	}
}
