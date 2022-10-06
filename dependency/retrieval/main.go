package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/joshuatcasey/libdependency/retrieve"
	"github.com/joshuatcasey/libdependency/upstream"
	"github.com/joshuatcasey/libdependency/versionology"
	"github.com/paketo-buildpacks/packit/v2/cargo"
)

type PyPiProductMetadataRaw struct {
	Releases map[string][]struct {
		PackageType string            `json:"packagetype"`
		URL         string            `json:"url"`
		UploadTime  string            `json:"upload_time_iso_8601"`
		Digests     map[string]string `json:"digests"`
	} `json:"releases"`
}

type PoetryRelease struct {
	version      *semver.Version
	SourceURL    string
	UploadTime   time.Time
	SourceSHA256 string
}

func (release PoetryRelease) Version() *semver.Version {
	return release.version
}

func getAllVersions() (versionology.VersionFetcherArray, error) {

	var poetryMetadata PyPiProductMetadataRaw
	err := upstream.GetAndUnmarshal("https://pypi.org/pypi/poetry/json", &poetryMetadata)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve new versions from upstream: %w", err)
	}

	var allVersions versionology.VersionFetcherArray

	for version, releasesForVersion := range poetryMetadata.Releases {
		for _, release := range releasesForVersion {
			if release.PackageType != "sdist" {
				continue
			}

			fmt.Printf("Parsing semver version %s\n", version)

			newVersion, err := semver.NewVersion(version)
			if err != nil {
				continue
			}

			uploadTime, err := time.Parse(time.RFC3339, release.UploadTime)
			if err != nil {
				return nil, fmt.Errorf("could not parse upload time '%s' as date for version %s: %w", release.UploadTime, version, err)
			}

			allVersions = append(allVersions, PoetryRelease{
				version:      newVersion,
				SourceSHA256: release.Digests["sha256"],
				SourceURL:    release.URL,
				UploadTime:   uploadTime,
			})
		}
	}

	return allVersions, nil
}

func generateMetadata(versionFetcher versionology.VersionFetcher) ([]versionology.Dependency, error) {
	version := versionFetcher.Version().String()
	poetryRelease, ok := versionFetcher.(PoetryRelease)
	if !ok {
		return nil, errors.New("expected a PoetryRelease")
	}

	configMetadataDependency := cargo.ConfigMetadataDependency{
		CPE:            fmt.Sprintf("cpe:2.3:a:python-poetry:poetry:%s:*:*:*:*:python:*:*", version),
		Checksum:       fmt.Sprintf("sha256:%s", poetryRelease.SourceSHA256),
		ID:             "poetry",
		Licenses:       retrieve.LookupLicenses(poetryRelease.SourceURL, upstream.DefaultDecompress),
		Name:           "Poetry",
		PURL:           retrieve.GeneratePURL("poetry", version, poetryRelease.SourceSHA256, poetryRelease.SourceURL),
		Source:         poetryRelease.SourceURL,
		SourceChecksum: fmt.Sprintf("sha256:%s", poetryRelease.SourceSHA256),
		Stacks:         []string{"*"},
		URI:            poetryRelease.SourceURL,
		Version:        version,
	}

	return []versionology.Dependency{{
		ConfigMetadataDependency: configMetadataDependency,
		SemverVersion:            versionFetcher.Version(),
	}}, nil
}

func main() {
	retrieve.NewMetadata("poetry", getAllVersions, generateMetadata)
}
