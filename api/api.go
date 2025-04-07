package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/gorilla/mux"
)

// REVIEW: Define a package level logger that will be correctly set in the call to New:
// var apiLogger *slog.Logger = slog.Default()

// REVIEW: Add a parameter for the logger.
func New() http.Handler {
	// REVIEW: Set the passed in logger to the package level apiLogger.
	router := mux.NewRouter()
	router.Handle("/package/{package}/{version}", http.HandlerFunc(packageHandler))
	return router
}

type npmPackageMetaResponse struct {
	Versions map[string]npmPackageResponse `json:"versions"`
}

type npmPackageResponse struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

type NpmPackageVersion struct {
	Name         string                        `json:"name"`
	Version      string                        `json:"version"`
	Dependencies map[string]*NpmPackageVersion `json:"dependencies"`
}

func packageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pkgName := vars["package"]
	pkgVersion := vars["version"]

	rootPkg := &NpmPackageVersion{Name: pkgName, Dependencies: make(map[string]*NpmPackageVersion)}
	// REVIEW: Declare the visited map[string]bool described in the comment above resolveDependencies
	// i.e., visited := make(map[string]bool) and pass the map to resolveDependencies.
	if err := resolveDependencies(rootPkg, pkgVersion); err != nil {
		// REVIEW: Use a log package instead of just printing errors to standard out.
		println(err.Error())
		// REVIEW: Use http package status codes instead of hardcoding the status.
		w.WriteHeader(500)
		return
	}

	stringified, err := json.MarshalIndent(rootPkg, "", "  ")
	if err != nil {
		// REVIEW: Use a log package instead of just printing errors to standard out.
		println(err.Error())
		// REVIEW: Use http package status codes instead of hardcoding the status.
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// REVIEW: Use http package status codes instead of hardcoding the status.
	w.WriteHeader(200)

	// Ignoring ResponseWriter errors
	// REVIEW: Don't ignore the errors, handle them properly.
	_, _ = w.Write(stringified)
}

// REVIEW: Circular dependencies could cause infinite recursion. Add a visited map[string]bool
// where the key is the name + "@" + concreteVersion. Then you need to add a check after the
// pkg.Version is set to see if we this key is in visited. If it is, stop processing.
func resolveDependencies(pkg *NpmPackageVersion, versionConstraint string) error {
	pkgMeta, err := fetchPackageMeta(pkg.Name)
	if err != nil {
		return err
	}
	concreteVersion, err := highestCompatibleVersion(versionConstraint, pkgMeta)
	if err != nil {
		return err
	}
	pkg.Version = concreteVersion

	npmPkg, err := fetchPackage(pkg.Name, pkg.Version)
	if err != nil {
		return err
	}
	for dependencyName, dependencyVersionConstraint := range npmPkg.Dependencies {
		dep := &NpmPackageVersion{Name: dependencyName, Dependencies: map[string]*NpmPackageVersion{}}
		pkg.Dependencies[dependencyName] = dep
		// REVIEW: Add the visited map to the resolveDependencies call below.
		if err := resolveDependencies(dep, dependencyVersionConstraint); err != nil {
			return err
		}
	}
	return nil
}

func highestCompatibleVersion(constraintStr string, versions *npmPackageMetaResponse) (string, error) {
	constraint, err := semver.NewConstraint(constraintStr)
	if err != nil {
		return "", err
	}
	filtered := filterCompatibleVersions(constraint, versions)
	sort.Sort(filtered)
	if len(filtered) == 0 {
		return "", errors.New("no compatible versions found")
	}
	return filtered[len(filtered)-1].String(), nil
}

func filterCompatibleVersions(constraint *semver.Constraints, pkgMeta *npmPackageMetaResponse) semver.Collection {
	var compatible semver.Collection
	for version := range pkgMeta.Versions {
		semVer, err := semver.NewVersion(version)
		if err != nil {
			continue
		}
		if constraint.Check(semVer) {
			compatible = append(compatible, semVer)
		}
	}
	return compatible
}

func fetchPackage(name, version string) (*npmPackageResponse, error) {
	resp, err := http.Get(fmt.Sprintf("https://registry.npmjs.org/%s/%s", name, version))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed npmPackageResponse

	// REVIEW: Unhandled errors here. Do something like the following:
	// err = json.Unmarshal(body, &parsed)
	//	if err != nil {
	//		return nil, fmt.Errorf("failed to unmarshal package data: %w", err)
	//	}
	_ = json.Unmarshal(body, &parsed)
	return &parsed, nil
}

func fetchPackageMeta(p string) (*npmPackageMetaResponse, error) {
	// REVIEW: Never check the status code of the HTTP Response we get back from NPM.
	// See next comment.
	resp, err := http.Get(fmt.Sprintf("https://registry.npmjs.org/%s", p))
	if err != nil {
		return nil, err
	}

	// REVIEW: Add something like this:
	//if resp.StatusCode != http.StatusOK {
	//	return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	//}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed npmPackageMetaResponse

	// REVIEW: No need to cast the body to a []byte. Already is that type.
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return nil, err
	}

	return &parsed, nil
}
