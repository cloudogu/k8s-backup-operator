Ginkgo testing — Guide (English)
================================

Purpose of this guide
---------------------
This document explains how to work with the BDD testing framework Ginkgo (commonly used with Gomega) in this project. Special attention is given to Suites — without a Suite you cannot run Ginkgo specs.

Prerequisites
-------------
- Go (version as required by `go.mod`)
- Ginkgo CLI (optional but recommended)

Installing the Ginkgo CLI
-------------------------
We recommend installing the Ginkgo CLI to generate boilerplate and run tests conveniently:

```bash
go install github.com/onsi/ginkgo/v2/ginkgo@latest
```

You can also run tests with `go test`.

First steps: bootstrap and suites
--------------------------------
Ginkgo uses a suite file per package to bridge Go's standard `testing` package and Ginkgo. The suite is mandatory: without it Ginkgo specs will not be registered and therefore not executed.

The Ginkgo CLI can generate a suite for you:

```bash
ginkgo bootstrap
```

This typically creates a `suite_test.go` file in the package with a simplified content like:

```go
package yourpackage_test

import (
	"testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestYourPackage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "YourPackage Suite")
}
```

Important: `RunSpecs` connects Ginkgo with `testing.T`. If you don't have this `Test...` function in a suite file, your Ginkgo specs will not run — therefore: have exactly one suite per package that contains Ginkgo specs.

Writing specs
-------------
You can generate a spec template with the CLI:

```bash
ginkgo generate my_feature
```

A typical spec file looks like this:

```go
var _ = Describe("MyFeature", func() {
	Context("in a certain context", func() {
		BeforeEach(func() {
			// setup
		})

		It("should do something", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(expected))
		})
	})
})
```

Key concepts
------------
- Describe / Context: grouping of specs
- It: single test case
- BeforeEach / AfterEach: setup/teardown for each spec
- BeforeAll / AfterAll: setup/teardown once per Describe block (use with care)

Gomega expectations
--------------------
Examples with Gomega:

```go
Expect(err).ToNot(HaveOccurred())
Expect(actual).To(Equal(expected))
Expect(list).To(ContainElement(item))
```

Running tests
-------------
With the Ginkgo CLI:

```bash
ginkgo -r
```

Useful options:
- `--randomizeAllSpecs` (randomize spec order)
- `--failFast` (stop on first failure)
- `--trace` (show stack traces)
- `--fail-on-pending` (treat pending tests as failures)

With go test:

```bash
go test ./... -v
```

CI note: Use `ginkgo -r` or `go test ./...` depending on your preference; ensure suite files are committed.

Common pitfalls and tips
-----------------------
- Missing suite: this is the most common cause when Ginkgo reports zero specs. Ensure the `Test...` suite function exists.
- Package name: test files should use the same package or `<pkg>_test`. Put the suite in the correct package.
- Race detector: `go test -race ./...` is useful, but be careful with test ordering and shared state in Ginkgo tests.

Example workflow
----------------
1. Generate suite once per package: `ginkgo bootstrap`
2. Generate specs: `ginkgo generate <name>` or create manually
3. Run tests locally: `ginkgo -r` or `go test ./...`
4. In CI: `ginkgo -r --fail-on-pending --randomizeAllSpecs` (or equivalent `go test`)

Further reading
---------------
- Ginkgo docs: https://onsi.github.io/ginkgo/
- Gomega docs: https://onsi.github.io/gomega/

Summary: Suites are mandatory
----------------------------
Without a suite file that contains a `Test...` function calling `RunSpecs`, Ginkgo specs will not run. Make sure every package using Ginkgo has a committed suite file.