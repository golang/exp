// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"golang.org/x/exp/vulndb/internal/audit"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/packages/packagestest"
)

// TODO(zpavlinovic): improve integration tests.

// goYamlVuln contains vulnerability info for github.com/go-yaml/yaml package.
var goYamlVuln string = `[{"ID":"GO-2020-0036","Published":"2021-04-14T12:00:00Z","Modified":"2021-04-14T12:00:00Z","Withdrawn":null,"Aliases":["CVE-2019-11254"],"Package":{"Name":"github.com/go-yaml/yaml","Ecosystem":"go"},"Details":"An attacker can craft malicious YAML which will consume significant\nsystem resources when Unmarshalled.\n","Affects":{"Ranges":[{"Type":"SEMVER","Introduced":"","Fixed":"v2.2.8+incompatible"}]},"References":[{"Type":"code review","URL":"https://github.com/go-yaml/yaml/pull/555"},{"Type":"fix","URL":"https://github.com/go-yaml/yaml/commit/53403b58ad1b561927d19068c655246f2db79d48"},{"Type":"misc","URL":"https://bugs.chromium.org/p/oss-fuzz/issues/detail?id=18496"}],"ecosystem_specific":{"Symbols":["yaml_parser_fetch_more_tokens"],"URL":"https://go.googlesource.com/vulndb/+/refs/heads/main/reports/GO-2020-0036.toml"}},{"ID":"GO-2021-0061","Published":"2021-04-14T12:00:00Z","Modified":"2021-04-14T12:00:00Z","Withdrawn":null,"Package":{"Name":"github.com/go-yaml/yaml","Ecosystem":"go"},"Details":"A maliciously crafted input can cause resource exhaustion due to\nalias chasing.\n","Affects":{"Ranges":[{"Type":"SEMVER","Introduced":"","Fixed":"v2.2.3+incompatible"}]},"References":[{"Type":"code review","URL":"https://github.com/go-yaml/yaml/pull/375"},{"Type":"fix","URL":"https://github.com/go-yaml/yaml/commit/bb4e33bf68bf89cad44d386192cbed201f35b241"}],"ecosystem_specific":{"Symbols":["decoder.unmarshal"],"URL":"https://go.googlesource.com/vulndb/+/refs/heads/main/reports/GO-2021-0061.toml"}}]`

// cryptoSSHVuln contains vulnerability info for golang.org/x/crypto/ssh.
var cryptoSSHVuln string = `[{"ID":"GO-2020-0012","Published":"2021-04-14T12:00:00Z","Modified":"2021-04-14T12:00:00Z","Withdrawn":null,"Aliases":["CVE-2020-9283"],"Package":{"Name":"golang.org/x/crypto/ssh","Ecosystem":"go"},"Details":"An attacker can craft an ssh-ed25519 or sk-ssh-ed25519@openssh.com public\nkey, such that the library will panic when trying to verify a signature\nwith it.\n","Affects":{"Ranges":[{"Type":"SEMVER","Introduced":"","Fixed":"v0.0.0-20200220183623-bac4c82f6975"}]},"References":[{"Type":"code review","URL":"https://go-review.googlesource.com/c/crypto/+/220357"},{"Type":"fix","URL":"https://github.com/golang/crypto/commit/bac4c82f69751a6dd76e702d54b3ceb88adab236"},{"Type":"misc","URL":"https://groups.google.com/g/golang-announce/c/3L45YRc91SY"}],"ecosystem_specific":{"Symbols":["parseED25519","ed25519PublicKey.Verify","parseSKEd25519","skEd25519PublicKey.Verify","NewPublicKey"],"URL":"https://go.googlesource.com/vulndb/+/refs/heads/main/reports/GO-2020-0012.toml"}},{"ID":"GO-2020-0013","Published":"2021-04-14T12:00:00Z","Modified":"2021-04-14T12:00:00Z","Withdrawn":null,"Aliases":["CVE-2017-3204"],"Package":{"Name":"golang.org/x/crypto/ssh","Ecosystem":"go"},"Details":"By default host key verification is disabled which allows for\nman-in-the-middle attacks against SSH clients if\n[ClientConfig.HostKeyCallback] is not set.\n","Affects":{"Ranges":[{"Type":"SEMVER","Introduced":"","Fixed":"v0.0.0-20170330155735-e4e2799dd7aa"}]},"References":[{"Type":"code review","URL":"https://go-review.googlesource.com/38701"},{"Type":"fix","URL":"https://github.com/golang/crypto/commit/e4e2799dd7aab89f583e1d898300d96367750991"},{"Type":"misc","URL":"https://github.com/golang/go/issues/19767"},{"Type":"misc","URL":"https://bridge.grumpy-troll.org/2017/04/golang-ssh-security/"}],"ecosystem_specific":{"Symbols":["NewClientConn"],"URL":"https://go.googlesource.com/vulndb/+/refs/heads/main/reports/GO-2020-0013.toml"}}]`

// k8sAPIServerVuln contains vulnerability info for k8s.io/apiextensions-apiserver/pkg/apiserver.
var k8sAPIServerVuln string = `[{"ID":"GO-2021-0062","Published":"2021-04-14T12:00:00Z","Modified":"2021-04-14T12:00:00Z","Withdrawn":null,"Aliases":["CVE-2019-11253"],"Package":{"Name":"k8s.io/apiextensions-apiserver/pkg/apiserver","Ecosystem":"go"},"Details":"A maliciously crafted YAML or JSON message can cause resource\nexhaustion.\n","Affects":{"Ranges":[{"Type":"SEMVER","Introduced":"","Fixed":"v0.17.0"}]},"References":[{"Type":"code review","URL":"https://github.com/kubernetes/kubernetes/pull/83261"},{"Type":"fix","URL":"https://github.com/kubernetes/apiextensions-apiserver/commit/9cfd100448d12f999fbf913ae5d4fef2fcd66871"},{"Type":"misc","URL":"https://github.com/kubernetes/kubernetes/issues/83253"},{"Type":"misc","URL":"https://gist.github.com/bgeesaman/0e0349e94cd22c48bf14d8a9b7d6b8f2"}],"ecosystem_specific":{"Symbols":["NewCustomResourceDefinitionHandler"],"URL":"https://go.googlesource.com/vulndb/+/refs/heads/main/reports/GO-2021-0062.toml"}}]`

// index for dbs containing some entries for each vuln package.
// The timestamp for package is set to random moment in the past.
var index string = `{
	"k8s.io/apiextensions-apiserver/pkg/apiserver": "2021-01-01T12:00:00.000000000-08:00",
	"golang.org/x/crypto/ssh": "2021-01-01T12:00:00.000000000-08:00",
	"github.com/go-yaml/yaml": "2021-01-01T12:00:00.000000000-08:00"
}`

var vulns = map[string]string{
	"github.com/go-yaml/yaml.json":                      goYamlVuln,
	"golang.org/x/crypto/ssh.json":                      cryptoSSHVuln,
	"k8s.io/apiextensions-apiserver/pkg/apiserver.json": k8sAPIServerVuln,
}

// addToLocalDb adds vuln for package p to local db at path db.
func addToLocalDb(db, p, vuln string) error {
	if err := os.MkdirAll(path.Join(db, filepath.Dir(p)), fs.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(path.Join(db, p))
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write([]byte(vuln))
	return nil
}

// addToServerDb adds vuln for package p to localhost server identified by its handler.
func addToServerDb(handler *http.ServeMux, p, vuln string) {
	handler.HandleFunc("/"+p, func(w http.ResponseWriter, req *http.Request) { fmt.Fprint(w, vuln) })
}

// envUpdate updates an environment e by setting the key to value.
func envUpdate(e []string, key, value string) []string {
	var nenv []string
	for _, kv := range e {
		if strings.HasPrefix(kv, key+"=") {
			nenv = append(nenv, key+"="+value)
		} else {
			nenv = append(nenv, kv)
		}
	}
	return nenv
}

// cmd type encapsulating a shell command and its context.
type cmd struct {
	dir  string
	env  []string
	name string
	args []string
}

// execAll executes a sequence of commands cmd. Exits on a first
// encountered error returning the error and the accumulated output.
func execAll(cmds []cmd) ([]byte, error) {
	var out []byte
	for _, c := range cmds {
		o, err := execCmd(c.dir, c.env, c.name, c.args...)
		out = append(out, o...)
		if err != nil {
			return o, err
		}
	}
	return out, nil
}

// execCmd runs the command name with arg in dir location with the env environment.
func execCmd(dir string, env []string, name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	cmd.Dir = dir
	cmd.Env = env
	return cmd.CombinedOutput()
}

// finding abstraction of Finding, for test purposes.
type finding struct {
	symbol   string
	traceLen int
}

func testFindings(finds []audit.Finding) []finding {
	var fs []finding
	for _, f := range finds {
		fs = append(fs, finding{symbol: f.Symbol, traceLen: len(f.Trace)})
	}
	return fs
}

func subset(finds1, finds2 []finding) bool {
	fs2 := make(map[finding]bool)
	for _, f := range finds2 {
		fs2[f] = true
	}

	for _, f := range finds1 {
		if !fs2[f] {
			return false
		}
	}
	return true
}

func TestHashicorpVault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	e := packagestest.Export(t, packagestest.Modules, []packagestest.Module{
		{
			Name: "foo",
		},
	})
	defer e.Cleanup()

	hashiVaultOkta := "github.com/hashicorp/vault/builtin/credential/okta"

	// Go get hashicorp-vault okta package v1.6.3.
	env := envUpdate(e.Config.Env, "GOPROXY", "https://proxy.golang.org,direct")
	if out, err := execCmd(e.Config.Dir, env, "go", "get", hashiVaultOkta+"@v1.6.3"); err != nil {
		t.Logf("failed to get %s: %s", hashiVaultOkta+"@v1.6.3", out)
		t.Fatal(err)
	}

	// run goaudit.
	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax | packages.NeedModule,
		Tests: false,
		Dir:   e.Config.Dir,
	}

	// Create a local filesystem db.
	dbPath := path.Join(e.Config.Dir, "db")
	addToLocalDb(dbPath, "index.json", index)
	// Create a local server db.
	sMux := http.NewServeMux()
	s := http.Server{Addr: ":8080", Handler: sMux}
	go func() { s.ListenAndServe() }()
	defer func() { s.Shutdown(context.Background()) }()
	addToServerDb(sMux, "index.json", index)

	for _, test := range []struct {
		source string
		// list of packages whose vulns should be addded to source
		toAdd []string
		want  []finding
	}{
		// test local db without yaml, which should result in no findings.
		{source: "file://" + dbPath, want: nil,
			toAdd: []string{"golang.org/x/crypto/ssh.json", "k8s.io/apiextensions-apiserver/pkg/apiserver.json"}},
		// add yaml to the local db, which should produce 2 findings.
		{source: "file://" + dbPath, toAdd: []string{"github.com/go-yaml/yaml.json"},
			want: []finding{
				{"github.com/go-yaml/yaml.decoder.unmarshal", 6},
				{"github.com/go-yaml/yaml.yaml_parser_fetch_more_tokens", 12}},
		},
		// repeat the similar experiment with a server db.
		{source: "http://localhost:8080", toAdd: []string{"k8s.io/apiextensions-apiserver/pkg/apiserver.json"}, want: nil},
		{source: "http://localhost:8080", toAdd: []string{"golang.org/x/crypto/ssh.json", "github.com/go-yaml/yaml.json"},
			want: []finding{
				{"github.com/go-yaml/yaml.decoder.unmarshal", 6},
				{"github.com/go-yaml/yaml.yaml_parser_fetch_more_tokens", 12}},
		},
	} {
		for _, add := range test.toAdd {
			if strings.HasPrefix(test.source, "file://") {
				addToLocalDb(dbPath, add, vulns[add])
			} else {
				addToServerDb(sMux, add, vulns[add])
			}
		}

		finds, err := run(cfg, []string{hashiVaultOkta}, false, []string{test.source})
		if err != nil {
			t.Fatal(err)
		}
		sort.SliceStable(finds, func(i int, j int) bool { return audit.FindingCompare(finds[i], finds[j]) })
		if fs := testFindings(finds); !subset(test.want, fs) {
			t.Errorf("want %v subset of findings; got %v", test.want, fs)
		}
	}
}

// isSecure checks if http resp was made over a secure connection.
func isSecure(resp *http.Response) bool {
	if resp.TLS == nil {
		return false
	}

	// Check the final URL scheme too for good measure.
	if resp.Request.URL.Scheme != "https" {
		return false
	}

	return true
}

// download fetches the content at url and stores it at destination location.
func download(url, destination string) error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !isSecure(resp) {
		return fmt.Errorf("insecure connection to %s", url)
	}

	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// TestKubernetes requires the following system dependencies:
//   - make, tar, unzip, and gcc.
// More information on installing kubernetes: https://github.com/kubernetes/kubernetes.
// Note that the whole installation will require roughly 5GB of disk.
func TestKubernetes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	e := packagestest.Export(t, packagestest.Modules, []packagestest.Module{
		{
			Name: "foo",
		},
	})
	defer e.Cleanup()

	// Environments and directories to build and download both k8s and go.
	env := envUpdate(e.Config.Env, "GOPROXY", "https://proxy.golang.org,direct")
	dir := e.Config.Dir
	k8sDir := path.Join(e.Config.Dir, "kubernetes-1.15.11")
	k8sEnv := envUpdate(env, "PATH", path.Join(e.Config.Dir, "go/bin")+":"+os.Getenv("PATH"))

	// Download kubernetes v1.15.11 and the go version 1.12 needed to build it.
	if err := download("https://github.com/kubernetes/kubernetes/archive/v1.15.11.zip", path.Join(dir, "v1.15.11")); err != nil {
		t.Fatal(err)
	}
	goZip := "go1.12.17." + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"
	if err := download("https://golang.org/dl/"+goZip, path.Join(dir, goZip)); err != nil {
		t.Fatal(err)
	}

	// Unzip k8s and go, and then build the k8s.
	if out, err := execAll([]cmd{
		{dir, env, "unzip", []string{"v1.15.11"}},
		{dir, env, "tar", []string{"-xf", goZip}},
		{k8sDir, k8sEnv, "make", nil},
	}); err != nil {
		t.Logf("failed to build k8s: %s", out)
		t.Fatal(err)
	}

	// Create a local filesystem db.
	dbPath := path.Join(e.Config.Dir, "db")
	addToLocalDb(dbPath, "index.json", index)
	// Create a local server db.
	sMux := http.NewServeMux()
	s := http.Server{Addr: ":8080", Handler: sMux}
	go func() { s.ListenAndServe() }()
	defer func() { s.Shutdown(context.Background()) }()
	addToServerDb(sMux, "index.json", index)

	// run goaudit.
	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax | packages.NeedModule,
		Tests: false,
		Dir:   path.Join(e.Config.Dir, "kubernetes-1.15.11"),
	}

	for _, test := range []struct {
		source string
		// list of packages whose vulns should be addded to source
		toAdd []string
		want  []finding
	}{
		// test local db with only apiserver vuln, which should result in a single finding.
		{source: "file://" + dbPath, toAdd: []string{"github.com/go-yaml/yaml.json", "k8s.io/apiextensions-apiserver/pkg/apiserver.json"},
			want: []finding{{"k8s.io/apiextensions-apiserver/pkg/apiserver.NewCustomResourceDefinitionHandler", 3}}},
		// add the rest of the vulnerabilites, resulting in more findings.
		{source: "file://" + dbPath, toAdd: []string{"golang.org/x/crypto/ssh.json"},
			want: []finding{
				{"golang.org/x/crypto/ssh.NewPublicKey", 1},
				{"k8s.io/apiextensions-apiserver/pkg/apiserver.NewCustomResourceDefinitionHandler", 3},
				{"golang.org/x/crypto/ssh.NewPublicKey", 4},
				{"golang.org/x/crypto/ssh.parseED25519", 9},
			}},
		// repeat similar experiment with a server db.
		{source: "http://localhost:8080", toAdd: []string{"github.com/go-yaml/yaml.json"}, want: nil},
		{source: "http://localhost:8080", toAdd: []string{"golang.org/x/crypto/ssh.json", "k8s.io/apiextensions-apiserver/pkg/apiserver.json"},
			want: []finding{
				{"golang.org/x/crypto/ssh.NewPublicKey", 1},
				{"k8s.io/apiextensions-apiserver/pkg/apiserver.NewCustomResourceDefinitionHandler", 3},
				{"golang.org/x/crypto/ssh.NewPublicKey", 4},
				{"golang.org/x/crypto/ssh.parseED25519", 9},
			}},
	} {
		for _, add := range test.toAdd {
			if strings.HasPrefix(test.source, "file://") {
				addToLocalDb(dbPath, add, vulns[add])
			} else {
				addToServerDb(sMux, add, vulns[add])
			}
		}

		finds, err := run(cfg, []string{"./..."}, false, []string{test.source})
		if err != nil {
			t.Fatal(err)
		}
		sort.SliceStable(finds, func(i int, j int) bool { return audit.FindingCompare(finds[i], finds[j]) })
		if fs := testFindings(finds); !subset(test.want, fs) {
			t.Errorf("want %v subset of findings; got %v", test.want, fs)
		}
	}
}
