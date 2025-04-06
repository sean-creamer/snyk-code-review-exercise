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

func New() http.Handler {
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

	rootPkg := &NpmPackageVersion{Name: pkgName, Dependencies: map[string]*NpmPackageVersion{}}
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

	// REVIEW: Unhandled errors here.
	_ = json.Unmarshal(body, &parsed)
	return &parsed, nil
}

func fetchPackageMeta(p string) (*npmPackageMetaResponse, error) {
	// REVIEW: We never check the status code here. A bad status code doesn't result in an err.
	resp, err := http.Get(fmt.Sprintf("https://registry.npmjs.org/%s", p))
	if err != nil {
		return nil, err
	}

	// REVIEW: Need something like below.
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
