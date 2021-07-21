package docker_registry

import (
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

type ResolveImplementationEntry struct {
	imagesRepoAddress string
	expectation       string
}

var _ = DescribeTable("ResolveImplementation", func(entry ResolveImplementationEntry) {
	resolvedImplementation, err := ResolveImplementation(entry.imagesRepoAddress, "")
	Ω(err).ShouldNot(HaveOccurred())

	Ω(resolvedImplementation).Should(Equal(entry.expectation))
},
	Entry("ecr", ResolveImplementationEntry{
		imagesRepoAddress: "123456789012.dkr.ecr.test.amazonaws.com/repo",
		expectation:       "ecr",
	}),
	Entry("acr", ResolveImplementationEntry{
		imagesRepoAddress: "test.azurecr.io/repo",
		expectation:       "acr",
	}),
	Entry("default", ResolveImplementationEntry{
		imagesRepoAddress: "localhost:5000/repo",
		expectation:       "default",
	}),
	Entry("dockerhub", ResolveImplementationEntry{
		imagesRepoAddress: "account/repo",
		expectation:       "dockerhub",
	}),
	Entry("gcr", ResolveImplementationEntry{
		imagesRepoAddress: "container.cloud.google.com/registry/repo",
		expectation:       "gcr",
	}),
	Entry("github", ResolveImplementationEntry{
		imagesRepoAddress: "docker.pkg.github.com/account/project/package",
		expectation:       "github",
	}),
	Entry("harbor", ResolveImplementationEntry{
		imagesRepoAddress: "harbor.company.com/project/repo",
		expectation:       "harbor",
	}),
	Entry("quay", ResolveImplementationEntry{
		imagesRepoAddress: "quay.io/account/repo",
		expectation:       "quay",
	}),
)
