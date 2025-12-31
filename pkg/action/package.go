/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package action

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Masterminds/semver/v3"
	"golang.org/x/term"
	"sigs.k8s.io/yaml"

	ci "helm.sh/helm/v4/pkg/chart"
	"helm.sh/helm/v4/pkg/chart/loader"
	chart "helm.sh/helm/v4/pkg/chart/v2"
	chartutil "helm.sh/helm/v4/pkg/chart/v2/util"
	"helm.sh/helm/v4/pkg/provenance"
)

// Package is the action for packaging a chart.
//
// It provides the implementation of 'helm package'.
type Package struct {
	Sign             bool
	Key              string
	Keyring          string
	PassphraseFile   string
	cachedPassphrase []byte
	Version          string
	AppVersion       string
	Destination      string
	DependencyUpdate bool

	RepositoryConfig      string
	RepositoryCache       string
	PlainHTTP             bool
	Username              string
	Password              string
	CertFile              string
	KeyFile               string
	CaFile                string
	InsecureSkipTLSVerify bool

	// ModTimeOverride to override all modification times in chart archives.
	ModTimeOverride time.Time
}

const (
	passPhraseFileStdin = "-"
)

// NewPackage creates a new Package object with the given configuration.
func NewPackage() *Package {
	return &Package{}
}

// Run executes 'helm package' against the given chart and returns the path to the packaged chart.
func (p *Package) Run(path string, _ map[string]interface{}) (string, error) {
	chrt, err := loader.LoadDir(path)
	if err != nil {
		return "", err
	}
	var ch *chart.Chart
	switch c := chrt.(type) {
	case *chart.Chart:
		ch = c
	case chart.Chart:
		ch = &c
	default:
		return "", errors.New("invalid chart apiVersion")
	}

	ac, err := ci.NewAccessor(ch)
	if err != nil {
		return "", err
	}

	// If version is set, modify the version.
	if p.Version != "" {
		ch.Metadata.Version = p.Version
	}

	if err := validateVersion(ch.Metadata.Version); err != nil {
		return "", err
	}

	if p.AppVersion != "" {
		ch.Metadata.AppVersion = p.AppVersion
	}

	if reqs := ac.MetaDependencies(); len(reqs) > 0 {
		if err := CheckDependencies(ch, reqs); err != nil {
			return "", err
		}
	}

	var dest string
	if p.Destination == "." {
		// Save to the current working directory.
		dest, err = os.Getwd()
		if err != nil {
			return "", err
		}
	} else {
		// Otherwise save to set destination
		dest = p.Destination
	}

	// Override the modification times (in both chart and its dependencies) for reproducible builds, if ModTimeOverride
	// is set.
	p.overrideAllModTimesInChart(ch)

	name, err := chartutil.Save(ch, dest)
	if err != nil {
		return "", fmt.Errorf("failed to save: %w", err)
	}

	if p.Sign {
		err = p.Clearsign(name)
	}

	return name, err
}

// validateVersion Verify that version is a Version, and error out if it is not.
func validateVersion(ver string) error {
	if _, err := semver.NewVersion(ver); err != nil {
		return err
	}
	return nil
}

// Clearsign signs a chart
func (p *Package) Clearsign(filename string) error {
	// Load keyring
	signer, err := provenance.NewFromKeyring(p.Keyring, p.Key)
	if err != nil {
		return err
	}

	passphraseFetcher := promptUser
	if p.PassphraseFile != "" {
		passphraseFetcher, err = p.passphraseFileFetcher(p.PassphraseFile, os.Stdin)
		if err != nil {
			return err
		}
	}

	if err := signer.DecryptKey(passphraseFetcher); err != nil {
		return err
	}

	// Load the chart archive to extract metadata
	chrt, err := loader.LoadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to load chart for signing: %w", err)
	}
	var ch *chart.Chart
	switch c := chrt.(type) {
	case *chart.Chart:
		ch = c
	case chart.Chart:
		ch = &c
	default:
		return errors.New("invalid chart apiVersion")
	}

	// Marshal chart metadata to YAML bytes
	metadataBytes, err := yaml.Marshal(ch.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal chart metadata: %w", err)
	}

	// Read the chart archive file
	archiveData, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read chart archive: %w", err)
	}

	// Use the generic provenance signing function
	sig, err := signer.ClearSign(archiveData, filepath.Base(filename), metadataBytes)
	if err != nil {
		return err
	}

	return os.WriteFile(filename+".prov", []byte(sig), 0644)
}

// promptUser implements provenance.PassphraseFetcher
func promptUser(name string) ([]byte, error) {
	fmt.Printf("Password for key %q >  ", name)
	// syscall.Stdin is not an int in all environments and needs to be coerced
	// into one there (e.g., Windows)
	pw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	return pw, err
}

func (p *Package) passphraseFileFetcher(passphraseFile string, stdin *os.File) (provenance.PassphraseFetcher, error) {
	// When reading from stdin we cache the passphrase here. If we are
	// packaging multiple charts, we reuse the cached passphrase. This
	// allows giving the passphrase once on stdin without failing with
	// complaints about stdin already being closed.
	//
	// An alternative to this would be to omit file.Close() for stdin
	// below and require the user to provide the same passphrase once
	// per chart on stdin, but that does not seem very user-friendly.

	if p.cachedPassphrase == nil {
		file, err := openPassphraseFile(passphraseFile, stdin)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		passphrase, _, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}
		p.cachedPassphrase = passphrase

		return func(_ string) ([]byte, error) {
			return passphrase, nil
		}, nil
	}

	return func(_ string) ([]byte, error) {
		return p.cachedPassphrase, nil
	}, nil
}

// overrideAllModTimesInChart overrides all modification times in the given chart and its dependencies to the provided
// time. The chart (and its dependencies) is modified in-place.
func (p *Package) overrideAllModTimesInChart(ch *chart.Chart) {
	// If the chart is nil or ModTimeOverride is zero, skip overriding the modification times in chart and its
	// dependencies.
	if ch == nil || p.ModTimeOverride.IsZero() {
		return
	}

	// Inline function to override all modification times in the given chart.
	overrideAllModTimesInChart := func(ch *chart.Chart, override time.Time) {
		// Override the chart modification time.
		ch.ModTime = override

		// Override the schema modification time for the chart.
		ch.SchemaModTime = override

		// Override the modification time for all files in the chart.
		for i := range ch.Files {
			ch.Files[i].ModTime = override
		}

		// Override the modification time for all the raw files in the chart.
		for i := range ch.Raw {
			ch.Raw[i].ModTime = override
		}

		// Override the modification time for all files in the templates.
		for i := range ch.Templates {
			ch.Templates[i].ModTime = override
		}
	}

	// Override the chart modification times.
	overrideAllModTimesInChart(ch, p.ModTimeOverride)

	// Override the modification times for all dependencies.
	chartDependencies := ch.Dependencies()
	for _, dep := range chartDependencies {
		overrideAllModTimesInChart(dep, p.ModTimeOverride)
	}
}

func openPassphraseFile(passphraseFile string, stdin *os.File) (*os.File, error) {
	if passphraseFile == passPhraseFileStdin {
		stat, err := stdin.Stat()
		if err != nil {
			return nil, err
		}
		if (stat.Mode() & os.ModeNamedPipe) == 0 {
			return nil, errors.New("specified reading passphrase from stdin, without input on stdin")
		}
		return stdin, nil
	}
	return os.Open(passphraseFile)
}
