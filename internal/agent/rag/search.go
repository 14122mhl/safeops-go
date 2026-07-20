// Package rag retrieves relevant local operational documents.
package rag

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Document is a bounded operational-document excerpt.
type Document struct {
	Path, Title, Excerpt string
	Score                int
}

// Searcher retrieves context without requiring an external vector database.
type Searcher interface {
	Search(context.Context, string) ([]Document, error)
}

// LocalSearcher performs deterministic lexical retrieval over configured paths.
type LocalSearcher struct {
	Paths                  []string
	MaxDocuments, MaxChars int
	Root                   string
}

// Search ranks documents by query-token overlap.
func (searcher LocalSearcher) Search(ctx context.Context, query string) ([]Document, error) {
	maxDocuments := searcher.MaxDocuments
	if maxDocuments <= 0 {
		maxDocuments = 3
	}
	maxChars := searcher.MaxChars
	if maxChars <= 0 {
		maxChars = 1200
	}
	queryTokens := tokens(query)
	paths, err := searcher.files(ctx)
	if err != nil {
		return nil, err
	}
	documents := make([]Document, 0, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		text, err := readBounded(path, maxChars*4)
		if err != nil {
			continue
		}
		score := overlap(queryTokens, tokens(text))
		if score == 0 {
			continue
		}
		excerpt := strings.TrimSpace(text)
		if len(excerpt) > maxChars {
			excerpt = excerpt[:maxChars]
		}
		documents = append(documents, Document{Path: searcher.displayPath(path), Title: title(path, text), Excerpt: excerpt, Score: score})
	}
	sort.Slice(documents, func(i, j int) bool {
		if documents[i].Score == documents[j].Score {
			return documents[i].Path < documents[j].Path
		}
		return documents[i].Score > documents[j].Score
	})
	if len(documents) > maxDocuments {
		documents = documents[:maxDocuments]
	}
	return documents, nil
}

// List returns all configured document paths without excerpts.
func (searcher LocalSearcher) List(ctx context.Context) ([]Document, error) {
	paths, err := searcher.files(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Document, 0, len(paths))
	for _, path := range paths {
		text, _ := readBounded(path, 4096)
		result = append(result, Document{Path: searcher.displayPath(path), Title: title(path, text)})
	}
	return result, nil
}

// PromptContext serializes retrieved documents for a bounded LLM prompt.
func PromptContext(documents []Document) string {
	var builder strings.Builder
	for index, document := range documents {
		if index > 0 {
			builder.WriteString("\n\n")
		}
		fmt.Fprintf(&builder, "Document: %s\nTitle: %s\nExcerpt:\n%s", document.Path, document.Title, document.Excerpt)
	}
	return builder.String()
}

func (searcher LocalSearcher) files(ctx context.Context) ([]string, error) {
	root := searcher.Root
	if root == "" {
		root = "."
	}
	seen := map[string]bool{}
	var result []string
	for _, configured := range searcher.Paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		path := configured
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			if supported(path) && !seen[path] {
				seen[path] = true
				result = append(result, path)
			}
			continue
		}
		err = filepath.WalkDir(path, func(candidate string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if !entry.IsDir() && supported(candidate) && !seen[candidate] {
				seen[candidate] = true
				result = append(result, candidate)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(result)
	return result, nil
}

func (searcher LocalSearcher) displayPath(path string) string {
	root := searcher.Root
	if root == "" {
		root = "."
	}
	if relative, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(relative, "..") {
		return filepath.ToSlash(relative)
	}
	return filepath.ToSlash(path)
}
func supported(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	return extension == ".md" || extension == ".txt"
}
func readBounded(path string, limit int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)))
	return string(data), err
}
func title(path, text string) string {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			value := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if value != "" {
				return value
			}
		}
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func tokens(value string) map[string]bool {
	result := map[string]bool{}
	var word, han []rune
	flushWord := func() {
		if len(word) > 1 {
			result[strings.ToLower(string(word))] = true
		}
		word = word[:0]
	}
	flushHan := func() {
		if len(han) > 1 {
			result[string(han)] = true
			for index := 0; index+1 < len(han); index++ {
				result[string(han[index:index+2])] = true
			}
		}
		han = han[:0]
	}
	for _, character := range value {
		switch {
		case unicode.Is(unicode.Han, character):
			flushWord()
			han = append(han, character)
		case unicode.IsLetter(character) || unicode.IsDigit(character) || character == '_':
			flushHan()
			word = append(word, unicode.ToLower(character))
		default:
			flushWord()
			flushHan()
		}
	}
	flushWord()
	flushHan()
	return result
}
func overlap(left, right map[string]bool) int {
	score := 0
	for token := range left {
		if right[token] {
			score++
		}
	}
	return score
}
