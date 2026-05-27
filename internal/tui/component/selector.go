package component

import "strings"

const defaultSelectorWindow = 4

type SelectorMatcher[T any] func(item T, query string) bool

type SelectorRenderer[T any] func(item T, selected bool) string

type SelectorSection[T any] func(item T) string

type SelectorConfig[T any] struct {
	Items   []T
	Match   SelectorMatcher[T]
	Render  SelectorRenderer[T]
	Section SelectorSection[T]
	Window  int
}

type Selector[T any] struct {
	items    []T
	matches  []T
	selected int
	query    string
	window   int

	match   SelectorMatcher[T]
	render  SelectorRenderer[T]
	section SelectorSection[T]
}

func NewSelector[T any](cfg SelectorConfig[T]) Selector[T] {
	window := cfg.Window
	if window <= 0 {
		window = defaultSelectorWindow
	}

	s := Selector[T]{
		items:   cfg.Items,
		match:   cfg.Match,
		render:  cfg.Render,
		section: cfg.Section,
		window:  window,
	}

	return s
}

func (s Selector[T]) View() string {
	if len(s.matches) == 0 {
		return ""
	}

	start, end := s.visibleRange()

	var (
		lines   []string
		prevSec string
		hasPrev bool
	)

	for idx := start; idx < end; idx++ {
		if s.section != nil {
			sec := s.section(s.matches[idx])
			if sec != "" && (!hasPrev || sec != prevSec) {
				lines = append(lines, sec)
			}

			prevSec = sec
			hasPrev = true
		}

		lines = append(lines, s.render(s.matches[idx], idx == s.selected))
	}

	return strings.Join(lines, "\n")
}

func (s *Selector[T]) SetItems(items []T) {
	s.items = items
	s.refilter()
}

func (s *Selector[T]) Filter(query string) {
	s.query = query
	s.refilter()
}

func (s *Selector[T]) Clear() {
	s.matches = nil
	s.selected = 0
	s.query = ""
}

func (s *Selector[T]) Previous() bool {
	if len(s.matches) == 0 {
		return false
	}

	s.selected = (s.selected - 1 + len(s.matches)) % len(s.matches)

	return true
}

func (s *Selector[T]) Next() bool {
	if len(s.matches) == 0 {
		return false
	}

	s.selected = (s.selected + 1) % len(s.matches)

	return true
}

func (s *Selector[T]) SelectWhere(pred func(T) bool) bool {
	for i, item := range s.matches {
		if pred(item) {
			s.selected = i

			return true
		}
	}

	return false
}

func (s Selector[T]) Empty() bool {
	return len(s.matches) == 0
}

func (s Selector[T]) Selected() (T, bool) {
	var zero T
	if len(s.matches) == 0 {
		return zero, false
	}

	return s.matches[s.selected], true
}

func (s *Selector[T]) refilter() {
	s.matches = s.matches[:0]
	for _, item := range s.items {
		if s.match(item, s.query) {
			s.matches = append(s.matches, item)
		}
	}

	if s.selected >= len(s.matches) {
		s.selected = 0
	}
}

func (s Selector[T]) visibleRange() (int, int) {
	if len(s.matches) <= s.window {
		return 0, len(s.matches)
	}

	start := s.selected - s.window/2
	start = max(start, 0)

	end := start + s.window
	if end > len(s.matches) {
		end = len(s.matches)
		start = end - s.window
	}

	return start, end
}
