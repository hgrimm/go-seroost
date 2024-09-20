package main

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type TermFreq map[string]int
type DocFreq map[string]int

type Doc struct {
	TF           TermFreq
	Count        int
	LastModified time.Time
}

type Docs map[string]*Doc

type Model struct {
	Docs Docs
	DF   DocFreq
}

func NewModel() *Model {
	return &Model{
		Docs: make(Docs),
		DF:   make(DocFreq),
	}
}

func (m *Model) RemoveDocument(filePath string) {
	if doc, exists := m.Docs[filePath]; exists {
		for term := range doc.TF {
			m.DF[term]--
			if m.DF[term] == 0 {
				delete(m.DF, term)
			}
		}
		delete(m.Docs, filePath)
	}
}

func (m *Model) RequiresReindexing(filePath string, lastModified time.Time) bool {
	if doc, exists := m.Docs[filePath]; exists {
		return doc.LastModified.Before(lastModified)
	}
	return true
}

func (m *Model) AddDocument(filePath string, lastModified time.Time, content []rune) {
	m.RemoveDocument(filePath)

	tf := make(TermFreq)
	count := 0
	lexer := NewLexer(content)
	for {
		token, ok := lexer.NextToken()
		if !ok {
			break
		}
		tf[token]++
		count++
	}

	for term := range tf {
		m.DF[term]++
	}

	m.Docs[filePath] = &Doc{
		TF:           tf,
		Count:        count,
		LastModified: lastModified,
	}
}

func computeTF(term string, doc *Doc) float64 {
	return float64(doc.TF[term]) / float64(doc.Count)
}

func computeIDF(term string, totalDocs int, df DocFreq) float64 {
	docFreq := float64(df[term])
	if docFreq == 0 {
		docFreq = 1
	}
	return math.Log10(float64(totalDocs) / docFreq)
}

func (m *Model) SearchQuery(query []rune) []SearchResult {
	lexer := NewLexer(query)
	var tokens []string
	for {
		token, ok := lexer.NextToken()
		if !ok {
			break
		}
		tokens = append(tokens, token)
	}

	// print tokens
	fmt.Printf("INFO: tokens: %v\n", tokens)

	var results []SearchResult
	for path, doc := range m.Docs {
		var rank float64
		for _, token := range tokens {
			rank += computeTF(token, doc) * computeIDF(token, len(m.Docs), m.DF)
		}
		if !math.IsNaN(rank) {
			results = append(results, SearchResult{Path: path, Rank: rank})
		}
	}

	// print results
	fmt.Printf("INFO: results: %v\n", results)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Rank > results[j].Rank
	})

	return results
}

type SearchResult struct {
	Path string  `json:"path"`
	Rank float64 `json:"rank"`
}
