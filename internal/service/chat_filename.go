package service

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const chatFileExtGroup = `txt|md|json|csv|log|pdf|doc|docx|xls|xlsx|ppt|pptx|jpg|jpeg|png|gif|bmp`

var (
	quotedFilenameRe = regexp.MustCompile(`["'“”]([\w\p{Han}\-\.\s]+\.(?:` + chatFileExtGroup + `))["'“”]`)
	plainFilenameRe  = regexp.MustCompile(`(?i)\b[\w\p{Han}\-\.]+\.(?:` + chatFileExtGroup + `)\b`)
)

// ExtractReferencedFilenames finds likely filenames in user text (quoted or bare word.ext).
func ExtractReferencedFilenames(text string) []string {
	if text == "" {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, m := range quotedFilenameRe.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		name := strings.TrimSpace(m[1])
		if name == "" {
			continue
		}
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	for _, m := range plainFilenameRe.FindAllString(text, -1) {
		name := strings.TrimSpace(m)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return filterSpuriousShorterFilenames(out)
}

// filterSpuriousShorterFilenames drops matches like "1.txt" when "notes 1.txt" is also present.
func filterSpuriousShorterFilenames(names []string) []string {
	if len(names) <= 1 {
		return names
	}
	keep := make([]bool, len(names))
	for i := range names {
		keep[i] = true
	}
	for i, a := range names {
		for _, b := range names {
			if len(b) <= len(a) {
				continue
			}
			if strings.HasSuffix(b, a) {
				prefix := b[:len(b)-len(a)]
				if len(prefix) == 0 {
					continue
				}
				r, _ := utf8.DecodeLastRuneInString(prefix)
				if r == utf8.RuneError {
					continue
				}
				if unicode.IsSpace(r) || r == '/' || r == '\\' || r == '"' || r == '\'' {
					keep[i] = false
					break
				}
			}
		}
	}
	var out []string
	for i, name := range names {
		if keep[i] {
			out = append(out, name)
		}
	}
	return out
}
