package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/productive-k3s/productive-k3s-cli/internal/bundles"
)

func compatibleAddonEntry() catalogEntry {
	entry := catalogEntry{Kind: "addon", Name: "nginx", Version: "0.1.0", SourceRevision: strings.Repeat("a", 40)}
	entry.Compatibility.Requires.Core.Contract = "artifact/v1"
	entry.Compatibility.Requires.Core.MinVersion = "0.9.6"
	entry.Compatibility.Requires.Core.MaxVersionExclusive = "0.10.0"
	entry.Compatibility.Requires.Kubernetes.Distros = []string{"k3s", "rke2"}
	return entry
}

func TestValidateCatalogEntryCompatibilityAcceptsCoreWindow(t *testing.T) {
	entry := compatibleAddonEntry()
	err := validateCatalogEntryCompatibility(entry, bundles.ReleaseManifest{CoreVersion: "0.9.6", InfraVersion: "0.9.65-0.9.6"})
	if err != nil {
		t.Fatalf("expected compatible entry: %v", err)
	}
}

func TestValidateCatalogEntryCompatibilityRejectsOldCore(t *testing.T) {
	entry := compatibleAddonEntry()
	err := validateCatalogEntryCompatibility(entry, bundles.ReleaseManifest{CoreVersion: "0.9.5", InfraVersion: "0.9.65-0.9.5"})
	if err == nil || !strings.Contains(err.Error(), "requires Core >=0.9.6 and <0.10.0") {
		t.Fatalf("expected actionable incompatibility, got %v", err)
	}
}

func TestValidateCatalogEntryCompatibilityParsesCompositeInfraRelease(t *testing.T) {
	entry := catalogEntry{Kind: "profile", Name: "multipass", Version: "0.1.0", SourceRevision: strings.Repeat("b", 40)}
	entry.Compatibility.Requires.Infra.Contract = "profile/v1"
	entry.Compatibility.Requires.Infra.MinEngineVersion = "0.9.65"
	entry.Compatibility.Requires.Infra.MaxEngineVersionExclusive = "0.10.0"
	entry.Compatibility.Requires.Core.MinVersion = "0.9.6"
	entry.Compatibility.Requires.Core.MaxVersionExclusive = "0.10.0"
	err := validateCatalogEntryCompatibility(entry, bundles.ReleaseManifest{CoreVersion: "0.9.6", InfraVersion: "0.9.65-0.9.6"})
	if err != nil {
		t.Fatalf("expected compatible profile: %v", err)
	}
}

func TestValidateCatalogEntryCompatibilityFailsClosed(t *testing.T) {
	entry := compatibleAddonEntry()
	entry.Compatibility.Requires.Core.Contract = "artifact/v2"
	if err := validateCatalogEntryCompatibility(entry, bundles.ReleaseManifest{CoreVersion: "0.9.6"}); err == nil {
		t.Fatal("expected unknown contract to fail")
	}
	entry = compatibleAddonEntry()
	entry.SourceRevision = ""
	if err := validateCatalogEntryCompatibility(entry, bundles.ReleaseManifest{CoreVersion: "0.9.6"}); err == nil {
		t.Fatal("expected missing provenance to fail")
	}
}

func TestParseCatalogEntriesReadsCompatibility(t *testing.T) {
	body := []byte(`entries:
  - id: nginx
    name: nginx
    kind: addon
    version: 0.1.0
    sourceRevision: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    compatibility:
      requires:
        core:
          contract: artifact/v1
          minVersion: 0.9.6
          maxVersionExclusive: 0.10.0
        kubernetes:
          distros: [k3s, rke2]
`)
	entries := parseCatalogEntries(body)
	if len(entries) != 1 || entries[0].Compatibility.Requires.Core.Contract != "artifact/v1" || len(entries[0].Compatibility.Requires.Kubernetes.Distros) != 2 {
		t.Fatalf("compatibility metadata was not parsed: %#v", entries)
	}
}

func TestVerifyCatalogSourceDigest(t *testing.T) {
	body := []byte("catalog")
	digest := sha256.Sum256(body)
	t.Setenv("PRODUCTIVE_K3S_CATALOG_URL_DEFAULT", "https://example.test/0.9.65/index.yaml")
	t.Setenv("PRODUCTIVE_K3S_CATALOG_SHA256_DEFAULT", hex.EncodeToString(digest[:]))
	if err := verifyCatalogSourceDigest(bundles.CatalogURLDefault(), body); err != nil {
		t.Fatalf("expected pinned catalog to validate: %v", err)
	}
	if err := verifyCatalogSourceDigest(bundles.CatalogURLDefault(), []byte("drift")); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected catalog checksum mismatch, got %v", err)
	}
	if err := verifyCatalogSourceDigest("https://example.test/development.yaml", []byte("custom")); err != nil {
		t.Fatalf("explicit custom catalog source should not inherit the release pin: %v", err)
	}
}

func TestVerifyCatalogSourceDigestFailsClosedOnInvalidPin(t *testing.T) {
	t.Setenv("PRODUCTIVE_K3S_CATALOG_URL_DEFAULT", "https://example.test/catalog.yaml")
	for _, digest := range []string{"short", strings.Repeat("z", sha256.Size*2)} {
		t.Setenv("PRODUCTIVE_K3S_CATALOG_SHA256_DEFAULT", digest)
		if err := verifyCatalogSourceDigest(bundles.CatalogURLDefault(), []byte("catalog")); err == nil || !strings.Contains(err.Error(), "configured catalog SHA-256 is invalid") {
			t.Fatalf("expected invalid pin %q to fail closed, got %v", digest, err)
		}
	}
}

func TestVerifyCatalogSourceDigestAllowsUnpinnedDevelopmentSource(t *testing.T) {
	t.Setenv("PRODUCTIVE_K3S_CATALOG_URL_DEFAULT", "https://example.test/catalog.yaml")
	t.Setenv("PRODUCTIVE_K3S_CATALOG_SHA256_DEFAULT", "")
	if err := verifyCatalogSourceDigest(bundles.CatalogURLDefault(), []byte("catalog")); err != nil {
		t.Fatalf("expected an explicitly unpinned development source to pass: %v", err)
	}
}

func TestParseStableSemver(t *testing.T) {
	version, err := parseStableSemver(" v1.2.3 ")
	if err != nil || version != (stableSemver{1, 2, 3}) {
		t.Fatalf("expected normalized stable semver, got %#v, %v", version, err)
	}
	for _, value := range []string{"1.2", "1.2.x", "1.-2.3"} {
		if _, err := parseStableSemver(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestValidateVersionWindowRejectsMalformedAndEmptyWindows(t *testing.T) {
	cases := []struct {
		name    string
		running string
		minimum string
		maximum string
		want    string
	}{
		{name: "running", running: "development", minimum: "0.9.6", maximum: "0.10.0", want: "running Core version is not comparable"},
		{name: "minimum", running: "0.9.6", minimum: "development", maximum: "0.10.0", want: "invalid Core minimum version"},
		{name: "maximum", running: "0.9.6", minimum: "0.9.6", maximum: "development", want: "invalid Core exclusive maximum version"},
		{name: "empty", running: "0.9.6", minimum: "0.10.0", maximum: "0.9.6", want: "invalid Core exclusive maximum version"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateVersionWindow("addon nginx 0.1.0", "Core", tc.running, tc.minimum, tc.maximum)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestSplitInfraRelease(t *testing.T) {
	infraVersion, coreVersion, err := splitInfraRelease("v0.9.65-0.9.6")
	if err != nil || infraVersion != "0.9.65" || coreVersion != "0.9.6" {
		t.Fatalf("unexpected composite release result: %q %q %v", infraVersion, coreVersion, err)
	}
	for _, value := range []string{"0.9.65", "bad-0.9.6", "0.9.65-bad"} {
		if _, _, err := splitInfraRelease(value); err == nil {
			t.Fatalf("expected invalid Infra release %q to fail", value)
		}
	}
}

func TestValidateProfileCompatibilityFailsClosed(t *testing.T) {
	entry := catalogEntry{Kind: "profile", Name: "multipass", Version: "0.1.0", SourceRevision: strings.Repeat("b", 40)}
	entry.Compatibility.Requires.Infra.Contract = "profile/v1"
	entry.Compatibility.Requires.Infra.MinEngineVersion = "0.9.65"
	entry.Compatibility.Requires.Infra.MaxEngineVersionExclusive = "0.10.0"
	entry.Compatibility.Requires.Core.MinVersion = "0.9.6"
	entry.Compatibility.Requires.Core.MaxVersionExclusive = "0.10.0"

	cases := []struct {
		name    string
		mutate  func(*catalogEntry)
		release bundles.ReleaseManifest
		want    string
	}{
		{name: "contract", mutate: func(value *catalogEntry) { value.Compatibility.Requires.Infra.Contract = "profile/v2" }, release: bundles.ReleaseManifest{InfraVersion: "0.9.65-0.9.6"}, want: "unsupported Infra contract"},
		{name: "release", mutate: func(*catalogEntry) {}, release: bundles.ReleaseManifest{InfraVersion: "0.9.65"}, want: "does not bind an exact Core version"},
		{name: "infra window", mutate: func(value *catalogEntry) { value.Compatibility.Requires.Infra.MinEngineVersion = "0.9.66" }, release: bundles.ReleaseManifest{InfraVersion: "0.9.65-0.9.6"}, want: "requires Infra >=0.9.66"},
		{name: "core window", mutate: func(value *catalogEntry) { value.Compatibility.Requires.Core.MinVersion = "0.9.7" }, release: bundles.ReleaseManifest{InfraVersion: "0.9.65-0.9.6"}, want: "requires Core >=0.9.7"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := entry
			tc.mutate(&candidate)
			err := validateCatalogEntryCompatibility(candidate, tc.release)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestDownloadCatalogEntryTGZValidatesArtifactAndChecksum(t *testing.T) {
	content := []byte("tgz-content")
	digest := sha256.Sum256(content)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	entry := compatibleAddonEntry()
	entry.Compatibility.Requires.Core.MinVersion = "0.9.5"
	entry.ArtifactType = "tgz"
	entry.ArtifactURL = server.URL + "/nginx.tgz"
	entry.ArtifactSHA256 = hex.EncodeToString(digest[:])
	deps := Dependencies{CacheDir: t.TempDir(), HTTPClient: server.Client()}

	path, err := downloadCatalogEntryTGZ(context.Background(), deps, entry)
	if err != nil {
		t.Fatalf("expected compatible artifact download: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(content) {
		t.Fatalf("unexpected downloaded artifact: %q, %v", got, err)
	}

	entry.ArtifactSHA256 = strings.Repeat("0", sha256.Size*2)
	path, err = downloadCatalogEntryTGZ(context.Background(), deps, entry)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected artifact checksum mismatch, got %q, %v", path, err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("checksum-invalid artifact should be removed, stat error: %v", statErr)
	}
}

func TestDownloadCatalogEntryTGZRequiresDownloadMetadata(t *testing.T) {
	entry := compatibleAddonEntry()
	entry.Compatibility.Requires.Core.MinVersion = "0.9.5"
	deps := Dependencies{CacheDir: t.TempDir(), HTTPClient: http.DefaultClient}

	if _, err := downloadCatalogEntryTGZ(context.Background(), deps, entry); err == nil || !strings.Contains(err.Error(), "does not expose a tgz artifact") {
		t.Fatalf("expected artifact type validation, got %v", err)
	}
	entry.ArtifactType = "tgz"
	if _, err := downloadCatalogEntryTGZ(context.Background(), deps, entry); err == nil || !strings.Contains(err.Error(), "downloadable tgz URL") {
		t.Fatalf("expected artifact URL validation, got %v", err)
	}
}
