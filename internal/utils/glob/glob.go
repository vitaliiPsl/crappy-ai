package glob

import (
	"path"
	"strings"

	"github.com/vitaliiPsl/crappy-ai/internal/utils"
)

const (
	pathSeparator     = "/"
	windowsSeparator  = "\\"
	currentDir        = "."
	recursiveWildcard = "**"
	rootSegment       = ""
	rootedPrefix      = "//"
	homePrefix        = "~/"
)

// Match tells whether input matches pattern by splitting both into path
// segments and comparing them with the supported wildcards (* and ? within a
// segment, [abc] / [a-z] classes, ** as a whole segment for zero-or-more
// segments).
func Match(pattern, input string) bool {
	pattern = normalize(pattern)
	input = normalize(input)

	return MatchSegments(split(pattern), split(input))
}

// MatchSegments tells whether input segments match pattern segments using the
// same wildcard semantics as Match. The caller is responsible for splitting and
// normalizing the segments for its domain.
func MatchSegments(pattern, input []string) bool {
	return matchSegments(pattern, input)
}

func normalize(s string) string {
	switch {
	case strings.HasPrefix(s, rootedPrefix):
		s = s[1:]
	case strings.HasPrefix(s, homePrefix):
		s = utils.ExpandHome(s)
	}

	return clean(s)
}

func clean(s string) string {
	s = strings.ReplaceAll(s, windowsSeparator, pathSeparator)
	if s == rootSegment {
		return currentDir
	}

	return path.Clean(s)
}

func split(s string) []string {
	if s == pathSeparator {
		return []string{rootSegment}
	}

	absolute := strings.HasPrefix(s, pathSeparator)
	if absolute {
		s = strings.TrimPrefix(s, pathSeparator)
	}

	parts := strings.Split(s, pathSeparator)
	if absolute {
		parts = append([]string{rootSegment}, parts...)
	}

	return parts
}

func matchSegments(pattern, input []string) bool {
	if len(pattern) == 0 {
		return len(input) == 0
	}

	if pattern[0] == recursiveWildcard {
		return matchDoubleStar(pattern, input)
	}

	if len(input) == 0 || !matchSegment(pattern[0], input[0]) {
		return false
	}

	return matchSegments(pattern[1:], input[1:])
}

func matchDoubleStar(pattern, input []string) bool {
	rest := pattern[1:]
	if len(rest) == 0 {
		return true
	}

	for i := 0; i <= len(input); i++ {
		if matchSegments(rest, input[i:]) {
			return true
		}
	}

	return false
}

func matchSegment(pattern, input string) bool {
	return matchSegmentAt([]rune(pattern), []rune(input), 0, 0)
}

func matchSegmentAt(pattern, input []rune, pi, ni int) bool {
	if pi == len(pattern) {
		return ni == len(input)
	}

	switch pattern[pi] {
	case '*':
		for i := ni; i <= len(input); i++ {
			if matchSegmentAt(pattern, input, pi+1, i) {
				return true
			}
		}

		return false
	case '?':
		return ni < len(input) && matchSegmentAt(pattern, input, pi+1, ni+1)
	case '[':
		ok, next := matchClass(pattern, input, pi, ni)

		return ok && matchSegmentAt(pattern, input, next, ni+1)
	default:
		return ni < len(input) &&
			pattern[pi] == input[ni] &&
			matchSegmentAt(pattern, input, pi+1, ni+1)
	}
}

func matchClass(pattern, input []rune, pi, ni int) (bool, int) {
	if ni >= len(input) {
		return false, pi + 1
	}

	end := pi + 1
	for end < len(pattern) && pattern[end] != ']' {
		end++
	}

	if end == len(pattern) {
		return pattern[pi] == input[ni], pi + 1
	}

	negated := false

	start := pi + 1
	if start < end && (pattern[start] == '!' || pattern[start] == '^') {
		negated = true
		start++
	}

	matched := false
	for i := start; i < end; i++ {
		if i+2 < end && pattern[i+1] == '-' {
			if pattern[i] <= input[ni] && input[ni] <= pattern[i+2] {
				matched = true
			}

			i += 2

			continue
		}

		if pattern[i] == input[ni] {
			matched = true
		}
	}

	if negated {
		matched = !matched
	}

	return matched, end + 1
}
