package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const sample = `default: backend

projects:
  - name: backend
    path: /tmp/projects/backend
    app: backend-app
    profiles:
      basic: [backend-app, backend-db]

  - name: admin
    path: /tmp/projects/frontend-admin

  - name: client
    path: /tmp/projects/frontend-client
    files: [docker-compose.dev.yml]

groups:
  system: [backend, admin, client]
`

func names(projects []Project) []string {
	list := make([]string, 0, len(projects))
	for _, p := range projects {
		list = append(list, p.Name)
	}
	return list
}

func TestParse(t *testing.T) {
	cfg, err := Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(cfg.Projects) != 3 {
		t.Fatalf("esperado 3 projetos, obtido %d", len(cfg.Projects))
	}
	backend := cfg.Projects[0]
	if backend.App != "backend-app" {
		t.Errorf("app não carregado: %q", backend.App)
	}
	if !reflect.DeepEqual(backend.Profiles["basic"], []string{"backend-app", "backend-db"}) {
		t.Errorf("perfil basic não carregado: %v", backend.Profiles)
	}
	if !reflect.DeepEqual(cfg.Projects[2].Files, []string{"docker-compose.dev.yml"}) {
		t.Errorf("files não carregado: %v", cfg.Projects[2].Files)
	}
	if !reflect.DeepEqual(cfg.Groups["system"], []string{"backend", "admin", "client"}) {
		t.Errorf("grupo não carregado: %v", cfg.Groups)
	}
	if cfg.Default != "backend" {
		t.Errorf("default não carregado: %q", cfg.Default)
	}
}

func TestParseInvalid(t *testing.T) {
	cases := map[string]string{
		"sem projetos":              "projects: []\n",
		"sem nome":                  "projects:\n  - path: /tmp/a\n",
		"sem path":                  "projects:\n  - name: a\n",
		"nome repetido":             "projects:\n  - name: a\n    path: /tmp/a\n  - name: A\n    path: /tmp/b\n",
		"chave inválida":            "projects:\n  - name: a\n    path: /tmp/a\n    container: x\n",
		"perfil vazio":              "projects:\n  - name: a\n    path: /tmp/a\n    profiles:\n      basic: []\n",
		"file vazio":                "projects:\n  - name: a\n    path: /tmp/a\n    files: ['']\n",
		"default inexistente":       "default: b\nprojects:\n  - name: a\n    path: /tmp/a\n",
		"grupo com membro ruim":     "projects:\n  - name: a\n    path: /tmp/a\ngroups:\n  g: [a, b]\n",
		"grupo vazio":               "projects:\n  - name: a\n    path: /tmp/a\ngroups:\n  g: []\n",
		"grupo com nome de projeto": "projects:\n  - name: a\n    path: /tmp/a\ngroups:\n  A: [a]\n",
		"nome com vírgula":          "projects:\n  - name: a,b\n    path: /tmp/a\n",
	}
	for name, data := range cases {
		if _, err := Parse([]byte(data)); err == nil {
			t.Errorf("%s: esperado erro", name)
		}
	}
}

func TestParseExpandsPaths(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("sem HOME")
	}
	cfg, err := Parse([]byte("projects:\n  - name: a\n    path: ~/projetos/a\n    files: [~/compose.yml, local.yml]\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if want := filepath.Join(home, "projetos", "a"); cfg.Projects[0].Path != want {
		t.Errorf("path = %q, esperado %q", cfg.Projects[0].Path, want)
	}
	want := []string{filepath.Join(home, "compose.yml"), "local.yml"}
	if !reflect.DeepEqual(cfg.Projects[0].Files, want) {
		t.Errorf("files = %v, esperado %v", cfg.Projects[0].Files, want)
	}
}

func TestSelectByFlag(t *testing.T) {
	cfg, err := Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	cases := []struct {
		raw  string
		want []string
	}{
		{raw: "admin", want: []string{"admin"}},
		{raw: "client, backend", want: []string{"client", "backend"}},
		{raw: "system", want: []string{"backend", "admin", "client"}},
		{raw: "SYSTEM,backend", want: []string{"backend", "admin", "client"}},
		{raw: "client,system", want: []string{"client", "backend", "admin"}},
		{raw: "admin,admin", want: []string{"admin"}},
	}
	for _, tc := range cases {
		selection, err := Select(cfg, tc.raw, "")
		if err != nil {
			t.Errorf("%q: %v", tc.raw, err)
			continue
		}
		if got := names(selection.Projects); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q: esperado %v, obtido %v", tc.raw, tc.want, got)
		}
		if selection.Origin != OriginFlag {
			t.Errorf("%q: origem %q", tc.raw, selection.Origin)
		}
	}
	if _, err := Select(cfg, "backend,inexistente", ""); err == nil || !strings.Contains(err.Error(), "inexistente") {
		t.Errorf("esperado erro para projeto inexistente, obtido %v", err)
	}
	if _, err := Select(cfg, " , ", ""); err == nil {
		t.Error("esperado erro para lista vazia")
	}
}

func TestSelectByCwdAndDefault(t *testing.T) {
	root := t.TempDir()
	backend := filepath.Join(root, "backend")
	nested := filepath.Join(backend, "packages", "admin")
	for _, dir := range []string{nested, filepath.Join(backend, "app")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	data := "default: system\nprojects:\n  - name: backend\n    path: " + backend + "\n  - name: admin\n    path: " + nested + "\ngroups:\n  system: [backend, admin]\n"
	cfg, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	cases := []struct {
		cwd    string
		want   []string
		origin string
	}{
		{cwd: filepath.Join(backend, "app"), want: []string{"backend"}, origin: OriginCwd},
		{cwd: nested, want: []string{"admin"}, origin: OriginCwd},
		{cwd: root, want: []string{"backend", "admin"}, origin: OriginDefault},
		{cwd: backend + "-outro", want: []string{"backend", "admin"}, origin: OriginDefault},
		{cwd: "", want: []string{"backend", "admin"}, origin: OriginDefault},
	}
	for _, tc := range cases {
		selection, err := Select(cfg, "", tc.cwd)
		if err != nil {
			t.Errorf("%q: %v", tc.cwd, err)
			continue
		}
		if got := names(selection.Projects); !reflect.DeepEqual(got, tc.want) || selection.Origin != tc.origin {
			t.Errorf("%q: esperado %v (%s), obtido %v (%s)", tc.cwd, tc.want, tc.origin, got, selection.Origin)
		}
	}

	selection, err := Select(cfg, "admin", filepath.Join(backend, "app"))
	if err != nil || !reflect.DeepEqual(names(selection.Projects), []string{"admin"}) {
		t.Errorf("--project deveria prevalecer sobre o diretório atual, obtido %v (%v)", names(selection.Projects), err)
	}
}

func TestSelectWithoutDefault(t *testing.T) {
	cfg, err := Parse([]byte("projects:\n  - name: a\n    path: /tmp/dm-inexistente-a\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Select(cfg, "", "/"); err == nil || !strings.Contains(err.Error(), "default") {
		t.Errorf("esperado erro pedindo -p ou default, obtido %v", err)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docker-manager.yml")
	if err := os.WriteFile(path, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, resolved, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if resolved != path {
		t.Errorf("resolved = %q, esperado %q", resolved, path)
	}
	if len(cfg.Projects) != 3 {
		t.Errorf("esperado 3 projetos, obtido %d", len(cfg.Projects))
	}
	if _, _, err := Load(filepath.Join(dir, "nao-existe.yml")); err == nil {
		t.Error("esperado erro para arquivo inexistente")
	}
}
