package service

import (
	"reflect"
	"testing"
)

func TestExtractReferencedFilenames(t *testing.T) {
	got := ExtractReferencedFilenames(`请总结 report.pdf 和 "notes 1.txt" 的内容`)
	want := []string{"notes 1.txt", "report.pdf"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
