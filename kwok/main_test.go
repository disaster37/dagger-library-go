package main

import (
	"strings"
	"testing"
)

func TestForgeDeploymentManifest(t *testing.T) {
	got := forgeDeploymentManifest("demo", "nginx:1.25", 3, "default")

	checks := map[string]string{
		"deployment name":    "name: demo",
		"namespace":          "namespace: default",
		"container image":    "image: nginx:1.25",
		"replicas":           "replicas: 3",
		"rollingUpdate type": "type: RollingUpdate",
		"maxUnavailable":     "maxUnavailable: 1",
		"maxSurge":           "maxSurge: 1",
	}
	for name, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("%s: manifest missing %q\n--- manifest ---\n%s", name, want, got)
		}
	}

	// Container name must equal the deployment name so `kubectl set image` works.
	if !strings.Contains(got, "- name: demo") {
		t.Errorf("container name should equal deployment name %q\n--- manifest ---\n%s", "demo", got)
	}
	// Exactly one container entry.
	if c := strings.Count(got, "- name: demo"); c != 1 {
		t.Errorf("expected exactly 1 container named demo, got %d", c)
	}
}

func TestIsDeploymentReady(t *testing.T) {
	cases := []struct {
		name      string
		status    string
		wantReady bool
		wantErr   bool
	}{
		{"fully ready", "3|3|3|1|1", true, false},
		{"ready<spec", "3|2|3|1|1", false, false},
		{"updated<spec", "3|3|2|1|1", false, false},
		{"observedGeneration<generation", "3|3|3|1|2", false, false},
		{"empty readyReplicas treated as 0", "3||3|1|1", false, false},
		{"all empty fields treated as 0", "0|0|0|0|0", false, false},
		{"replicas=0 edge", "0|0|0|1|1", false, false},
		{"malformed line too few fields", "3|3|3", false, true},
		{"malformed line too many fields", "3|3|3|1|1|1", false, true},
		{"non-numeric field", "3|abc|3|1|1", false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ready, err := isDeploymentReady(tc.status)
			if (err != nil) != tc.wantErr {
				t.Fatalf("isDeploymentReady(%q) err = %v, wantErr = %v", tc.status, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if ready != tc.wantReady {
				t.Errorf("isDeploymentReady(%q) = %v, want %v", tc.status, ready, tc.wantReady)
			}
		})
	}
}

func TestValidateK8sName(t *testing.T) {
	valid := []string{"a", "itest", "my-cluster", "cluster-1", "a1b2c3"}
	for _, v := range valid {
		if err := validateK8sName("test name", v); err != nil {
			t.Errorf("validateK8sName(%q) unexpected error: %v", v, err)
		}
	}

	// Every invalid case carries either shell/YAML metacharacters, path
	// separators, a leading dash (flag injection) or breaks the DNS-1123
	// grammar; all must be rejected (CWE-20).
	invalid := []string{
		"",                      // empty
		" ",                     // whitespace
		"My-Cluster",            // uppercase
		"-cluster",              // leading dash (kubectl flag injection)
		"cluster-",              // trailing dash
		"clu ster",              // space (shell word splitting)
		"cluster;id",            // shell command separator
		"$(curl evil)",          // shell command substitution
		"cluster`id`",           // backtick substitution
		"cluster|nc",            // pipe
		"cluster&&id",           // AND list
		"../etc",                // path traversal
		"a/b",                   // path separator
		"a.b",                   // dot
		"cluster\nname",         // newline (YAML injection)
		strings.Repeat("a", 64), // longer than 63 chars
	}
	for _, v := range invalid {
		if err := validateK8sName("test name", v); err == nil {
			t.Errorf("validateK8sName(%q) expected error, got nil", v)
		}
	}
}

func TestValidateImageRef(t *testing.T) {
	valid := []string{
		"nginx",
		"nginx:1.25",
		"library/nginx:latest",
		"registry.k8s.io/kwok/cluster:v0.6.1-k8s.v1.30.4",
		"my.registry:5000/org/img:v1.2.3",
		"nginx@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	for _, v := range valid {
		if err := validateImageRef("test image", v); err != nil {
			t.Errorf("validateImageRef(%q) unexpected error: %v", v, err)
		}
	}

	invalid := []string{
		"",                    // empty
		"nginx;id",            // shell command separator
		"nginx$(id)",          // command substitution
		"nginx`id`",           // backtick substitution
		"nginx|nc evil 4444",  // pipe
		"nginx && id",         // AND list / space
		"nginx\n  evil: yaml", // newline YAML injection
		"nginx'image",         // quote
		`nginx"image`,         // double quote
		"nginx>out",           // redirection
		"-nginx",              // leading dash (flag injection)
	}
	for _, v := range invalid {
		if err := validateImageRef("test image", v); err == nil {
			t.Errorf("validateImageRef(%q) expected error, got nil", v)
		}
	}
}

func TestValidateCIDR(t *testing.T) {
	valid := []string{"10.44.0.0/16", "10.45.0.0/16", "192.168.0.0/24", "2001:db8::/32"}
	for _, v := range valid {
		if err := validateCIDR("test cidr", v); err != nil {
			t.Errorf("validateCIDR(%q) unexpected error: %v", v, err)
		}
	}

	invalid := []string{
		"",                 // empty
		"10.44.0.0",        // missing prefix
		"not-a-cidr",       // garbage
		"10.44.0.0/16; id", // shell injection attempt
		"$(id)",            // command substitution
		"10.44.0.0/33",     // invalid prefix length
	}
	for _, v := range invalid {
		if err := validateCIDR("test cidr", v); err == nil {
			t.Errorf("validateCIDR(%q) expected error, got nil", v)
		}
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"plain":      `'plain'`,
		"with space": `'with space'`,
		"it's":       `'it'\''s'`,
		"$(evil)":    `'$(evil)'`,
		"; rm -rf /": `'; rm -rf /'`,
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewValidation(t *testing.T) {
	// Security (CWE-20, CWE-78, CWE-22, CWE-400): New must reject cluster
	// names that could inject into the startup shell script or traverse the
	// cache mount path, and node counts that could exhaust the container.
	// Validation happens before any Dagger API call, so no engine is needed.
	invalidNames := []string{"../evil", "a b", "Cluster;id", "$(id)", "-flag", ""}
	for _, name := range invalidNames {
		if _, err := New(name, 0, ""); err == nil {
			t.Errorf("New(%q) expected validation error, got nil", name)
		}
	}
	if _, err := New("itest", maxNodes+1, ""); err == nil {
		t.Errorf("New with nodes=%d expected validation error, got nil", maxNodes+1)
	}
	if _, err := New("itest", 1, ""); err != nil {
		t.Errorf("New with valid arguments should not fail: %v", err)
	}
}

func TestForgeServerScript(t *testing.T) {
	t.Run("defaults single node", func(t *testing.T) {
		m := &Kwok{Name: "itest", Nodes: 1}
		got := m.forgeServerScript("", "")

		mustContain := []string{
			"itest",
			"--kube-apiserver-port 6443",
			"--kube-apiserver-insecure-port 0",
			"--wait 120s",
			"-c /kwok/config.yaml",
			"kwokctl --name itest kubectl get --raw=/readyz",
			"kwokctl scale node node --name itest --replicas 1",
			"is already being served by another service instance",
			`sed -i "s#https://[0-9.]*:6443#https://127.0.0.1:6443#"`,
			"sed -i",
			"kubeconfig.yaml",
			"exec kwokctl logs kwok-controller -f --name itest",
			"command -v kwokctl",
		}
		for _, want := range mustContain {
			if !strings.Contains(got, want) {
				t.Errorf("script missing %q\n--- script ---\n%s", want, got)
			}
		}

		// No CIDR extra-args when params empty.
		if strings.Contains(got, "--extra-args") {
			t.Errorf("script should not contain --extra-args when CIDR params empty\n--- script ---\n%s", got)
		}
	})

	t.Run("zero nodes defaults to one", func(t *testing.T) {
		m := &Kwok{Name: "itest", Nodes: 0}
		got := m.forgeServerScript("", "")

		// kwok creates no nodes by default, so the script must always scale
		// to at least one node.
		if !strings.Contains(got, "kwokctl scale node node --name itest --replicas 1") {
			t.Errorf("script should scale to 1 node when Nodes <= 1\n--- script ---\n%s", got)
		}
	})

	t.Run("multi node scaling", func(t *testing.T) {
		m := &Kwok{Name: "itest", Nodes: 3}
		got := m.forgeServerScript("", "")

		if !strings.Contains(got, "kwokctl scale node node --name itest --replicas 3") {
			t.Errorf("script missing scale line with --name\n--- script ---\n%s", got)
		}
	})

	t.Run("cluster cidr wiring", func(t *testing.T) {
		m := &Kwok{Name: "itest", Nodes: 1}
		got := m.forgeServerScript("10.44.0.0/16", "")

		if !strings.Contains(got, "--extra-args kube-controller-manager=cluster-cidr=10.44.0.0/16") {
			t.Errorf("script missing cluster-cidr extra-args\n--- script ---\n%s", got)
		}
		if !strings.Contains(got, "--extra-args kube-controller-manager=allocate-node-cidrs=true") {
			t.Errorf("script missing allocate-node-cidrs extra-args\n--- script ---\n%s", got)
		}
	})

	t.Run("service cidr wiring", func(t *testing.T) {
		m := &Kwok{Name: "itest", Nodes: 1}
		got := m.forgeServerScript("", "10.45.0.0/16")

		if !strings.Contains(got, "--extra-args kube-apiserver=service-cluster-ip-range=10.45.0.0/16") {
			t.Errorf("script missing apiserver service-cluster-ip-range extra-args\n--- script ---\n%s", got)
		}
		if !strings.Contains(got, "--extra-args kube-controller-manager=service-cluster-ip-range=10.45.0.0/16") {
			t.Errorf("script missing controller-manager service-cluster-ip-range extra-args\n--- script ---\n%s", got)
		}
	})
}
