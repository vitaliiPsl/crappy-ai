package glob

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchLiteralPaths(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"exact relative", "src/main.go", "src/main.go", true},
		{"exact absolute", "/tmp/project/main.go", "/tmp/project/main.go", true},
		{"different file", "src/main.go", "src/app.go", false},
		{"different depth", "src/main.go", "src/pkg/main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.path); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchSegmentWildcards(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"star within segment", "src/*.go", "src/main.go", true},
		{"star does not cross slash", "src/*.go", "src/pkg/main.go", false},
		{"star can be empty", "src/main*.go", "src/main.go", true},
		{"question within segment", "src/file?.go", "src/file1.go", true},
		{"question requires one character", "src/file?.go", "src/file.go", false},
		{"character class", "src/file[0-9].go", "src/file7.go", true},
		{"character class mismatch", "src/file[0-9].go", "src/filex.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.path); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchRecursiveDoubleStar(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"recursive suffix matches root", "/tmp/project/**", "/tmp/project", true},
		{"recursive suffix matches child", "/tmp/project/**", "/tmp/project/main.go", true},
		{"recursive suffix matches deep child", "/tmp/project/**", "/tmp/project/pkg/app/main.go", true},
		{"recursive suffix rejects sibling", "/tmp/project/**", "/tmp/other/main.go", false},
		{"recursive prefix matches zero segments", "**/*.go", "main.go", true},
		{"recursive prefix matches many segments", "**/*.go", "src/pkg/main.go", true},
		{"recursive middle matches zero segments", "src/**/testdata/*.json", "src/testdata/input.json", true},
		{"recursive middle matches many segments", "src/**/testdata/*.json", "src/pkg/api/testdata/input.json", true},
		{"recursive middle keeps suffix", "src/**/testdata/*.json", "src/pkg/api/testdata/input.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.path); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchSegments(t *testing.T) {
	tests := []struct {
		name    string
		pattern []string
		input   []string
		want    bool
	}{
		{"star matches one segment", []string{"*", "example", "com"}, []string{"api", "example", "com"}, true},
		{"star rejects extra segment", []string{"*", "example", "com"}, []string{"v1", "api", "example", "com"}, false},
		{"double star matches many segments", []string{"**", "example", "com"}, []string{"v1", "api", "example", "com"}, true},
		{"wildcard inside segment", []string{"api-*", "example", "com"}, []string{"api-v1", "example", "com"}, true},
		{"question inside segment", []string{"ex?mple", "com"}, []string{"example", "com"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchSegments(tt.pattern, tt.input); got != tt.want {
				t.Errorf("MatchSegments(%v, %v) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchNormalizesSeparatorsAndDotSegments(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"backslash path separators", `src/**/*.go`, `src\pkg\main.go`, true},
		{"dot segment in path", "src/*.go", "src/./main.go", true},
		{"dot dot segment in path", "src/*.go", "src/pkg/../main.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.path); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchCharacterClassEdges(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"negated bang class matches outside range", "file[!0-9].go", "filex.go", true},
		{"negated bang class rejects range", "file[!0-9].go", "file7.go", false},
		{"negated caret class matches outside range", "file[^0-9].go", "filex.go", true},
		{"negated caret class rejects range", "file[^0-9].go", "file7.go", false},
		{"malformed class is literal bracket", "file[abc.go", "file[abc.go", true},
		{"malformed class does not act as wildcard", "file[abc.go", "filea.go", false},
		{"empty class does not match", "file[].go", "file].go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.path); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchAbsoluteAndRootPaths(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"absolute pattern does not match relative path", "/tmp/*.go", "tmp/main.go", false},
		{"relative pattern does not match absolute path", "tmp/*.go", "/tmp/main.go", false},
		{"root matches root", "/", "/", true},
		{"root does not match child", "/", "/tmp", false},
		{"recursive root matches root", "/**", "/", true},
		{"recursive root matches child", "/**", "/tmp/project/main.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.path); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchDoubleStarOnlyRecursiveAsWholeSegment(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"double star inside segment is regular stars", "src**/main.go", "srcpkg/main.go", true},
		{"double star inside segment does not cross slash", "src**/main.go", "src/pkg/main.go", false},
		{"double star suffix inside segment is regular stars", "**.go", "main.go", true},
		{"double star suffix inside segment does not cross slash", "**.go", "src/main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.path); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchCleanedTrailingSlashes(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"literal trailing slash is cleaned", "src/", "src", true},
		{"path trailing slash is cleaned", "src", "src/", true},
		{"recursive trailing slash is cleaned", "src/**/", "src/pkg", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.path); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchNormalizesRootedAndHomePrefixes(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("home dir unavailable: %v", err)
	}

	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"rooted prefix on pattern matches absolute path", "//tmp/project/**", "/tmp/project/main.go", true},
		{"rooted prefix on pattern rejects sibling", "//tmp/project/**", "/tmp/other/main.go", false},
		{"rooted prefix on path is stripped", "/tmp/project/main.go", "//tmp/project/main.go", true},
		{"home prefix on pattern expands", "~/notes/**", filepath.Join(home, "notes", "draft.md"), true},
		{"home prefix on path expands", filepath.Join(home, "notes", "draft.md"), "~/notes/draft.md", true},
		{"home prefix mismatch", "~/notes/**", filepath.Join(home, "other", "draft.md"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.path); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchUnicodeAndEmptyInputs(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"question matches one rune", "?.go", "å.go", true},
		{"star matches unicode runes", "*.go", "λambda.go", true},
		{"empty pattern matches empty path after cleaning", "", "", true},
		{"empty pattern does not match non-empty path", "", "src", false},
		{"non-empty pattern does not match empty path", "src", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.path); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}
