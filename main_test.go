//go:build conformance

// Conformance suite: needs envtest binaries (TEST_ASSET_*) and a real zone (TEST_ZONE_NAME).

package main

import (
	"os"
	"testing"

	dns "github.com/cert-manager/cert-manager/test/acme"
)

var (
	zone = os.Getenv("TEST_ZONE_NAME")
)

func TestRunsSuite(t *testing.T) {
	fixture := dns.NewFixture(&yandexCloudDNSSolver{},
		dns.SetResolvedZone(zone),
		dns.SetAllowAmbientCredentials(false),
		dns.SetManifestPath("testdata/yandex-cloud-dns"),
		dns.SetStrict(true),
	)

	fixture.RunConformance(t)
}
