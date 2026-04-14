package rag

import "fmt"

func GenerateIndexName(filename string) string {
	return fmt.Sprintf("rag_idx_%s", filename)
}

func GenerateIndexNamePrefix(filename string) string {
	return fmt.Sprintf("rag_docs:%s:", filename)
}
