package specs

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Books", func() {
	It("should be a novel", func() {
		Expect(5).To(Equal(5))
	})
})
