Ginkgo Testing — Anleitung (Deutsch)
=================================

Ziel dieser Anleitung
---------------------
Dieses Dokument beschreibt, wie im Projekt mit dem BDD-Testframework Ginkgo (häufig zusammen mit Gomega) gearbeitet wird. Besondere Aufmerksamkeit gilt dem Umgang mit Suites — ohne Suite können keine Tests ausgeführt werden.

Voraussetzungen
---------------
- Go (Version entsprechend `go.mod`)
- Ginkgo CLI (optional, aber sehr hilfreich)

Installation der Ginkgo CLI
--------------------------
Empfohlen ist die Installation der Ginkgo-CLI, um Boilerplate zu erzeugen und Tests komfortabel lokal und rekursiv auszuführen:

```bash
go install github.com/onsi/ginkgo/v2/ginkgo@latest
```

Alternativ laufen Tests auch mit `go test`.

Erste Schritte: Bootstrap und Suites
----------------------------------
Ginkgo benutzt eine Suite-Datei pro Paket, die eine Verbindung zwischen dem Standard-`testing`-Framework von Go und Ginkgo herstellt. Diese Suite ist zwingend erforderlich: ohne sie werden die Ginkgo-Spezifikationen nicht registriert und folglich nicht ausgeführt.

Die Ginkgo-CLI kann eine Suite automatisch generieren:

```bash
ginkgo bootstrap
```

Dies erzeugt typischerweise eine Datei namens `suite_test.go` im jeweiligen Paket mit folgendem (vereinfachtem) Inhalt:

```go
package deinpackage_test

import (
	"testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeinPackage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DeinPackage Suite")
}
```

Wichtig: `RunSpecs` verbindet Ginkgo mit `testing.T`. Ohne diese `Test...`-Funktion in der Suite werden keine Ginkgo-Tests ausgeführt — daher die Regel: jede Test-Datei mit Ginkgo-Spezifikationen benötigt eine Suite (pro Paket genügt eine).

Tests erstellen
---------------
Mit der Ginkgo-CLI lässt sich eine Beispiel-Spezifikation erzeugen:

```bash
ginkgo generate my_feature
```

Eine typische Spec-Datei enthält:

```go
var _ = Describe("MyFeature", func() {
	Context("in einem bestimmten Kontext", func() {
		BeforeEach(func() {
			// Setup
		})

		It("sollte etwas tun", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(expected))
		})
	})
})
```

Wichtige Konzepte
------------------
- Describe / Context: Gruppieren von Specs
- It: Einzelner Testfall
- BeforeEach / AfterEach: Setup/TearDown für jede Spec
- BeforeAll / AfterAll: Setup/TearDown einmal pro Describe-Block (vorsichtig verwenden)

Gomega-Erwartungen
-------------------
Beispiele mit Gomega:

```go
Expect(err).ToNot(HaveOccurred())
Expect(actual).To(Equal(expected))
Expect(list).To(ContainElement(item))
```

Testausführung
---------------
Mit Ginkgo-CLI:

```bash
ginkgo -r
```

Optionen, die hilfreich sind:
- `--randomizeAllSpecs` (Zufällige Reihenfolge)
- `--failFast` (bei erstem Fehler abbrechen)
- `--trace` (Stacktraces zeigen)
- `--fail-on-pending` (Pending-Tests als Fehler behandeln)

Mit go test:

```bash
go test ./... -v
```

Hinweis für CI: Nutzt `ginkgo -r` oder `go test ./...` je nach Präferenz; achtet darauf, dass die Suite-Dateien committed sind.

Fehlerquellen und Tipps
-----------------------
- Keine Suite vorhanden: Wird häufig übersehen. Wenn `ginkgo` oder `go test` keine Specs ausführt, prüft zuerst, ob die `Test...`-Funktion (Suite) vorhanden ist.
- Paketname: Testdateien sollten das selbe Paket oder `<pkg>_test` verwenden. Die Suite sollte im passenden Paket liegen.
- Race-Detector: `go test -race ./...` ist nützlich, bei manchen Ginkgo-Setups muss man aber auf die Reihenfolge und Shared-State achten.

Beispiel-Workflow
------------------
1. Suite erzeugen (einmal pro Paket): `ginkgo bootstrap`
2. Specs erzeugen: `ginkgo generate <name>` oder manuell anlegen
3. Tests lokal ausführen: `ginkgo -r` oder `go test ./...`
4. In CI: `ginkgo -r --fail-on-pending --randomizeAllSpecs` (oder entsprechend `go test`)

Weiterführende Links
--------------------
- Ginkgo Homepage / Docs: https://onsi.github.io/ginkgo/
- Gomega: https://onsi.github.io/gomega/

Kurzfassung: Suites sind obligatorisch
-----------------------------------
Ohne die Suite-Datei mit einer `Test...`-Funktion, die `RunSpecs` aufruft, laufen Ginkgo-Spezifikationen nicht. Achte also bei jedem Paket mit Ginkgo-Tests darauf, dass eine Suite vorhanden und committed ist.