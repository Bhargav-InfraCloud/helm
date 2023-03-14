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

package cli

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"helm.sh/helm/v3/internal/version"
)

func TestSetNamespace(t *testing.T) {
	settings := New()

	if settings.namespace != "" {
		t.Errorf("Expected empty namespace, got %s", settings.namespace)
	}

	settings.SetNamespace("testns")
	if settings.namespace != "testns" {
		t.Errorf("Expected namespace testns, got %s", settings.namespace)
	}

}

func TestEnvSettings(t *testing.T) {
	tests := []struct {
		name string

		// input
		args    string
		envvars map[string]string

		// expected values
		ns, kcontext  string
		debug         bool
		maxhistory    int
		kubeAsUser    string
		kubeAsGroups  []string
		kubeCaFile    string
		kubeInsecure  bool
		kubeTLSServer string
		burstLimit    int
	}{
		{
			name:       "defaults",
			ns:         "default",
			maxhistory: defaultMaxHistory,
			burstLimit: defaultBurstLimit,
		},
		{
			name:          "with flags set",
			args:          "--debug --namespace=myns --kube-as-user=poro --kube-as-group=admins --kube-as-group=teatime --kube-as-group=snackeaters --kube-ca-file=/tmp/ca.crt --burst-limit 100 --kube-insecure-skip-tls-verify=true --kube-tls-server-name=example.org",
			ns:            "myns",
			debug:         true,
			maxhistory:    defaultMaxHistory,
			burstLimit:    100,
			kubeAsUser:    "poro",
			kubeAsGroups:  []string{"admins", "teatime", "snackeaters"},
			kubeCaFile:    "/tmp/ca.crt",
			kubeTLSServer: "example.org",
			kubeInsecure:  true,
		},
		{
			name:          "with envvars set",
			envvars:       map[string]string{"HELM_DEBUG": "1", "HELM_NAMESPACE": "yourns", "HELM_KUBEASUSER": "pikachu", "HELM_KUBEASGROUPS": ",,,operators,snackeaters,partyanimals", "HELM_MAX_HISTORY": "5", "HELM_KUBECAFILE": "/tmp/ca.crt", "HELM_BURST_LIMIT": "150", "HELM_KUBEINSECURE_SKIP_TLS_VERIFY": "true", "HELM_KUBETLS_SERVER_NAME": "example.org"},
			ns:            "yourns",
			maxhistory:    5,
			burstLimit:    150,
			debug:         true,
			kubeAsUser:    "pikachu",
			kubeAsGroups:  []string{"operators", "snackeaters", "partyanimals"},
			kubeCaFile:    "/tmp/ca.crt",
			kubeTLSServer: "example.org",
			kubeInsecure:  true,
		},
		{
			name:          "with flags and envvars set",
			args:          "--debug --namespace=myns --kube-as-user=poro --kube-as-group=admins --kube-as-group=teatime --kube-as-group=snackeaters --kube-ca-file=/my/ca.crt --burst-limit 175 --kube-insecure-skip-tls-verify=true --kube-tls-server-name=example.org",
			envvars:       map[string]string{"HELM_DEBUG": "1", "HELM_NAMESPACE": "yourns", "HELM_KUBEASUSER": "pikachu", "HELM_KUBEASGROUPS": ",,,operators,snackeaters,partyanimals", "HELM_MAX_HISTORY": "5", "HELM_KUBECAFILE": "/tmp/ca.crt", "HELM_BURST_LIMIT": "200", "HELM_KUBEINSECURE_SKIP_TLS_VERIFY": "true", "HELM_KUBETLS_SERVER_NAME": "example.org"},
			ns:            "myns",
			debug:         true,
			maxhistory:    5,
			burstLimit:    175,
			kubeAsUser:    "poro",
			kubeAsGroups:  []string{"admins", "teatime", "snackeaters"},
			kubeCaFile:    "/my/ca.crt",
			kubeTLSServer: "example.org",
			kubeInsecure:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer resetEnv()()

			for k, v := range tt.envvars {
				os.Setenv(k, v)
			}

			flags := pflag.NewFlagSet("testing", pflag.ContinueOnError)

			settings := New()
			settings.AddFlags(flags)
			flags.Parse(strings.Split(tt.args, " "))

			if settings.Debug != tt.debug {
				t.Errorf("expected debug %t, got %t", tt.debug, settings.Debug)
			}
			if settings.Namespace() != tt.ns {
				t.Errorf("expected namespace %q, got %q", tt.ns, settings.Namespace())
			}
			if settings.KubeContext != tt.kcontext {
				t.Errorf("expected kube-context %q, got %q", tt.kcontext, settings.KubeContext)
			}
			if settings.MaxHistory != tt.maxhistory {
				t.Errorf("expected maxHistory %d, got %d", tt.maxhistory, settings.MaxHistory)
			}
			if tt.kubeAsUser != settings.KubeAsUser {
				t.Errorf("expected kAsUser %q, got %q", tt.kubeAsUser, settings.KubeAsUser)
			}
			if !reflect.DeepEqual(tt.kubeAsGroups, settings.KubeAsGroups) {
				t.Errorf("expected kAsGroups %+v, got %+v", len(tt.kubeAsGroups), len(settings.KubeAsGroups))
			}
			if tt.kubeCaFile != settings.KubeCaFile {
				t.Errorf("expected kCaFile %q, got %q", tt.kubeCaFile, settings.KubeCaFile)
			}
			if tt.burstLimit != settings.BurstLimit {
				t.Errorf("expected BurstLimit %d, got %d", tt.burstLimit, settings.BurstLimit)
			}
			if tt.kubeInsecure != settings.KubeInsecureSkipTLSVerify {
				t.Errorf("expected kubeInsecure %t, got %t", tt.kubeInsecure, settings.KubeInsecureSkipTLSVerify)
			}
			if tt.kubeTLSServer != settings.KubeTLSServerName {
				t.Errorf("expected kubeTLSServer %q, got %q", tt.kubeTLSServer, settings.KubeTLSServerName)
			}
		})
	}
}

func TestEnvOrBool(t *testing.T) {
	const envName = "TEST_ENV_OR_BOOL"
	tests := []struct {
		name     string
		env      string
		val      string
		def      bool
		expected bool
	}{
		{
			name:     "unset with default false",
			def:      false,
			expected: false,
		},
		{
			name:     "unset with default true",
			def:      true,
			expected: true,
		},
		{
			name:     "blank env with default false",
			env:      envName,
			def:      false,
			expected: false,
		},
		{
			name:     "blank env with default true",
			env:      envName,
			def:      true,
			expected: true,
		},
		{
			name:     "env true with default false",
			env:      envName,
			val:      "true",
			def:      false,
			expected: true,
		},
		{
			name:     "env false with default true",
			env:      envName,
			val:      "false",
			def:      true,
			expected: false,
		},
		{
			name:     "env fails parsing with default true",
			env:      envName,
			val:      "NOT_A_BOOL",
			def:      true,
			expected: true,
		},
		{
			name:     "env fails parsing with default false",
			env:      envName,
			val:      "NOT_A_BOOL",
			def:      false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				t.Cleanup(func() {
					os.Unsetenv(tt.env)
				})
				os.Setenv(tt.env, tt.val)
			}
			actual := envBoolOr(tt.env, tt.def)
			if actual != tt.expected {
				t.Errorf("expected result %t, got %t", tt.expected, actual)
			}
		})
	}
}

func TestUserAgentHeaderInK8sRESTClientConfig(t *testing.T) {
	defer resetEnv()()

	settings := New()
	restConfig, err := settings.RESTClientGetter().ToRESTConfig()
	if err != nil {
		t.Fatal(err)
	}

	expectedUserAgent := version.GetUserAgent()
	if restConfig.UserAgent != expectedUserAgent {
		t.Errorf("expected User-Agent header %q in K8s REST client config, got %q", expectedUserAgent, restConfig.UserAgent)
	}
}

func resetEnv() func() {
	origEnv := os.Environ()

	// ensure any local envvars do not hose us
	for e := range New().EnvVars() {
		os.Unsetenv(e)
	}

	return func() {
		for _, pair := range origEnv {
			kv := strings.SplitN(pair, "=", 2)
			os.Setenv(kv[0], kv[1])
		}
	}
}

func TestEnvSettings_BackupKubeConfig(t *testing.T) {
	var (
		testDataDir        = `testdata/`
		kubeConfigFilename = testDataDir + "kubeconfig"
	)

	type fields struct {
		KubeConfig     string
		helmConfigHome string
	}

	type toggles struct {
		wantErr               bool
		cleanUpTestKubeConfig bool
	}

	type testCase struct {
		name    string
		fields  fields
		toggles toggles
	}

	tests := []testCase{
		{
			name: "Backup kube config",
			fields: fields{
				KubeConfig:     testDataDir + `valid-kubeconfig-no-contexts`,
				helmConfigHome: testDataDir,
			},
			toggles: toggles{
				cleanUpTestKubeConfig: true,
			},
		},
		{
			name: "Failure missing input kube config file",
			fields: fields{
				KubeConfig:     testDataDir + `missing-kubeconfig`,
				helmConfigHome: testDataDir,
			},
			toggles: toggles{
				wantErr: true,
			},
		},
		{
			name: "Failure invalid destination path",
			fields: fields{
				KubeConfig:     testDataDir + `valid-kubeconfig-no-contexts`,
				helmConfigHome: testDataDir + `non-existing-dir/`,
			},
			toggles: toggles{
				wantErr: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &EnvSettings{
				KubeConfig: tt.fields.KubeConfig,
			}

			t.Setenv(`HELM_CONFIG_HOME`, tt.fields.helmConfigHome)

			err := s.BackupKubeConfig()
			if (err != nil) != tt.toggles.wantErr {
				t.Errorf("EnvSettings.BackupKubeConfig() error = %v, wantErr %v",
					err, tt.toggles.wantErr)
			}

			if !tt.toggles.wantErr && s.KubeConfig != kubeConfigFilename {
				t.Errorf("kube config path not updated after backup, want = %s, got = %s",
					kubeConfigFilename, s.KubeConfig)
			}

			if tt.toggles.cleanUpTestKubeConfig {
				err = os.Remove(kubeConfigFilename)
				if err != nil {
					t.Errorf("failed to delete %q: %v", kubeConfigFilename, err)
				}
			}
		})
	}
}
