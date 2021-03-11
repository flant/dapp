package path_matcher

func NewStubPathMatcher(basePath string) *StubPathMatcher {
	return &StubPathMatcher{SimplePathMatcher: NewSimplePathMatcher(basePath, nil)}
}

// StubPathMatcher returns false when matching paths
type StubPathMatcher struct {
	*SimplePathMatcher
}

func (m *StubPathMatcher) IsDirOrSubmodulePathMatched(_ string) bool {
	return false
}

func (m *StubPathMatcher) IsPathMatched(_ string) bool {
	return false
}

func (m *StubPathMatcher) ShouldGoThrough(_ string) bool {
	return false
}

func (m *StubPathMatcher) ID() string {
	return m.String()
}

func (m *StubPathMatcher) String() string {
	return "none"
}
