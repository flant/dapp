package docker_registry

import (
	"github.com/google/go-containerregistry/pkg/name"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

type ParseReferencePartsEntry struct {
	reference   string
	expectation referenceParts
}

var _ = DescribeTable("Api_ParseReferenceParts", func(entry ParseReferencePartsEntry) {
	parts, err := (&api{}).ParseReferenceParts(entry.reference)
	Ω(err).ShouldNot(HaveOccurred())
	Ω(parts).Should(Equal(entry.expectation))
},
	Entry("account/project", ParseReferencePartsEntry{
		reference: "account/project",
		expectation: referenceParts{
			registry:   name.DefaultRegistry,
			repository: "account/project",
			tag:        name.DefaultTag,
		},
	}),
	Entry("repo", ParseReferencePartsEntry{
		reference: "repo",
		expectation: referenceParts{
			registry:   name.DefaultRegistry,
			repository: "library/repo",
			tag:        name.DefaultTag,
		},
	}),
	Entry("registry.com/repo", ParseReferencePartsEntry{
		reference: "registry.com/repo",
		expectation: referenceParts{
			registry:   "registry.com",
			repository: "repo",
			tag:        name.DefaultTag,
		},
	}),
	Entry("registry.com/repo:tag", ParseReferencePartsEntry{
		reference: "registry.com/repo:tag",
		expectation: referenceParts{
			registry:   "registry.com",
			repository: "repo",
			tag:        "tag",
		},
	}),
	Entry("registry.com/repo:tag@sha256:db6697a61d5679b7ca69dbde3dad6be0d17064d5b6b0e9f7be8d456ebb337209", ParseReferencePartsEntry{
		reference: "registry.com/repo:tag@sha256:db6697a61d5679b7ca69dbde3dad6be0d17064d5b6b0e9f7be8d456ebb337209",
		expectation: referenceParts{
			registry:   "registry.com",
			repository: "repo",
			tag:        "tag",
			digest:     "sha256:db6697a61d5679b7ca69dbde3dad6be0d17064d5b6b0e9f7be8d456ebb337209",
		},
	}),
)
