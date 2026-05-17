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

// Match tells whether name matches pattern by splitting both into path
// segments and comparing them with the supported wildcards (* and ? within a
// segment, [abc] / [a-z] classes, ** as a whole segment for zero-or-more
// segments).
func Match(pattern, name string) bool {
	pattern = normalize(pattern)
	name = normalize(name)

	return matchSegments(split(pattern), split(name))
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

func matchSegments(pattern, name []string) bool {
	if len(pattern) == 0 {
		return len(name) == 0
	}

	if pattern[0] == recursiveWildcard {
		return matchDoubleStar(pattern, name)
	}

	if len(name) == 0 || !matchSegment(pattern[0], name[0]) {
		return false
	}

	return matchSegments(pattern[1:], name[1:])
}

func matchDoubleStar(pattern, name []string) bool {
	rest := pattern[1:]
	if len(rest) == 0 {
		return true
	}

	for i := 0; i <= len(name); i++ {
		if matchSegments(rest, name[i:]) {
			return true
		}
	}

	return false
}

func matchSegment(pattern, name string) bool {
	return matchSegmentAt([]rune(pattern), []rune(name), 0, 0)
}

func matchSegmentAt(pattern, name []rune, pi, ni int) bool {
	if pi == len(pattern) {
		return ni == len(name)
	}

	switch pattern[pi] {
	case '*':
		for i := ni; i <= len(name); i++ {
			if matchSegmentAt(pattern, name, pi+1, i) {
				return true
			}
		}

		return false
	case '?':
		return ni < len(name) && matchSegmentAt(pattern, name, pi+1, ni+1)
	case '[':
		ok, next := matchClass(pattern, name, pi, ni)

		return ok && matchSegmentAt(pattern, name, next, ni+1)
	default:
		return ni < len(name) &&
			pattern[pi] == name[ni] &&
			matchSegmentAt(pattern, name, pi+1, ni+1)
	}
}

func matchClass(pattern, name []rune, pi, ni int) (bool, int) {
	if ni >= len(name) {
		return false, pi + 1
	}

	end := pi + 1
	for end < len(pattern) && pattern[end] != ']' {
		end++
	}

	if end == len(pattern) {
		return pattern[pi] == name[ni], pi + 1
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
			if pattern[i] <= name[ni] && name[ni] <= pattern[i+2] {
				matched = true
			}

			i += 2

			continue
		}

		if pattern[i] == name[ni] {
			matched = true
		}
	}

	if negated {
		matched = !matched
	}

	return matched, end + 1
}
