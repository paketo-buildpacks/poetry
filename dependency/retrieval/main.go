package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/joshuatcasey/libdependency/retrieve"
	"github.com/joshuatcasey/libdependency/versionology"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/vacation"
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
	response, err := http.DefaultClient.Get("https://pypi.org/pypi/poetry/json")
	if err != nil {
		return nil, fmt.Errorf("could not get project metadata: %w", err)
	}

	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response: %w", err)
	}

	var poetryMetadata PyPiProductMetadataRaw
	err = json.Unmarshal(body, &poetryMetadata)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal project metadata: %w", err)
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
		Licenses:       retrieve.LookupLicenses(poetryRelease.SourceURL, defaultDecompress),
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

func defaultDecompress(artifact io.Reader, destination string) error {
	archive := vacation.NewArchive(artifact)

	err := archive.StripComponents(1).Decompress(destination)
	if err != nil {
		return fmt.Errorf("failed to decompress source file: %w", err)
	}

	return nil
}
