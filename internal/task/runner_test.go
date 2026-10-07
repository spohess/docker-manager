package task

import (
	"reflect"
	"strings"
	"testing"

	"docker-manager/internal/config"
)

var (
	backend = config.Project{
		Name:     "backend",
		Path:     "/tmp/backend",
		App:      "backend-app",
		Profiles: map[string][]string{"basic": {"backend-app", "backend-db"}, "queue": {"backend-app", "backend-redis"}},
	}
	admin  = config.Project{Name: "admin", Path: "/tmp/admin"}
	client = config.Project{Name: "client", Path: "/tmp/client", App: "client-app", Files: []string{"docker-compose.dev.yml"}}
)

func TestUpStep(t *testing.T) {
	cases := []struct {
		name    string
		project config.Project
		opts    Options
		want    []string
	}{
		{name: "perfil basic", project: backend, want: []string{"up", "-d", "backend-app", "backend-db"}},
		{name: "full", project: backend, opts: Options{Full: true}, want: []string{"up", "-d"}},
		{name: "outro perfil com build", project: backend, opts: Options{Profile: "queue", Build: true}, want: []string{"up", "-d", "--build", "backend-app", "backend-redis"}},
		{name: "sem perfil", project: admin, want: []string{"up", "-d"}},
		{name: "perfil ausente no projeto", project: admin, opts: Options{Profile: "queue"}, want: []string{"up", "-d"}},
	}
	for _, tc := range cases {
		got := upStep(tc.project, tc.opts)
		if !reflect.DeepEqual(got.args, tc.want) {
			t.Errorf("%s: esperado %v, obtido %v", tc.name, tc.want, got.args)
		}
	}
}

func TestProfileSteps(t *testing.T) {
	if got := stopStep(backend, Options{}).args; !reflect.DeepEqual(got, []string{"stop", "backend-app", "backend-db"}) {
		t.Errorf("stop: %v", got)
	}
	if got := restartStep(backend, Options{Full: true}).args; !reflect.DeepEqual(got, []string{"restart"}) {
		t.Errorf("restart: %v", got)
	}
	if got := downStep(backend, Options{}).args; !reflect.DeepEqual(got, []string{"down"}) {
		t.Errorf("down: %v", got)
	}
}

func TestReversed(t *testing.T) {
	got := joinNames(reversed([]config.Project{backend, admin, client}))
	if got != "client, admin, backend" {
		t.Errorf("ordem reversa inesperada: %s", got)
	}
}

func TestValidate(t *testing.T) {
	up, _ := lookup("up")
	down, _ := lookup("down")
	app, _ := lookup("app")
	cases := []struct {
		name    string
		def     definition
		opts    Options
		wantErr string
	}{
		{name: "up válido", def: up, opts: Options{Full: true, Build: true}},
		{name: "app com argumentos", def: app, opts: Options{Args: []string{"ls"}}},
		{name: "posicional no up", def: up, opts: Options{Args: []string{"redis"}}, wantErr: "argumento inesperado"},
		{name: "full com profile", def: up, opts: Options{Full: true, Profile: "queue"}, wantErr: "não podem ser usados juntos"},
		{name: "full no down", def: down, opts: Options{Full: true}, wantErr: "--full"},
		{name: "profile no app", def: app, opts: Options{Profile: "queue"}, wantErr: "--profile"},
		{name: "build no down", def: down, opts: Options{Build: true}, wantErr: "--build"},
	}
	for _, tc := range cases {
		err := validate(tc.def, tc.opts)
		if tc.wantErr == "" {
			if err != nil {
				t.Errorf("%s: %v", tc.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%s: esperado erro contendo %q, obtido %v", tc.name, tc.wantErr, err)
		}
	}
}

func TestCheckProfile(t *testing.T) {
	if err := checkProfile([]config.Project{admin, backend}, "queue"); err != nil {
		t.Errorf("perfil existente em um dos projetos: %v", err)
	}
	if err := checkProfile([]config.Project{admin, backend}, "queu"); err == nil {
		t.Error("esperado erro para perfil inexistente")
	}
}

func TestAppProject(t *testing.T) {
	project, err := appProject([]config.Project{backend, admin})
	if err != nil || project.Name != "backend" {
		t.Errorf("esperado backend, obtido %q (%v)", project.Name, err)
	}
	if _, err := appProject([]config.Project{admin}); err == nil || !strings.Contains(err.Error(), "nenhum") {
		t.Errorf("esperado erro sem app, obtido %v", err)
	}
	if _, err := appProject([]config.Project{backend, client}); err == nil || !strings.Contains(err.Error(), "mais de um") {
		t.Errorf("esperado erro com mais de um app, obtido %v", err)
	}
	if _, err := appTarget([]config.Project{backend}, Options{}); err == nil {
		t.Error("esperado erro sem comando")
	}
}

func TestDirectTargets(t *testing.T) {
	logs, err := logsTarget([]config.Project{client}, Options{Args: []string{"--tail=10", "client-app"}})
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if want := []string{"compose", "-f", "docker-compose.dev.yml", "logs", "-f", "--tail=10", "client-app"}; !reflect.DeepEqual(logs.args, want) || logs.dir != client.Path {
		t.Errorf("logs: esperado %v em %s, obtido %v em %s", want, client.Path, logs.args, logs.dir)
	}

	compose, err := composeTarget([]config.Project{backend}, Options{Args: []string{"pull"}})
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	if want := []string{"compose", "pull"}; !reflect.DeepEqual(compose.args, want) {
		t.Errorf("compose: esperado %v, obtido %v", want, compose.args)
	}

	if _, err := logsTarget([]config.Project{backend, admin}, Options{}); err == nil || !strings.Contains(err.Error(), "único projeto") {
		t.Errorf("esperado erro com mais de um projeto, obtido %v", err)
	}
	if _, err := composeTarget([]config.Project{backend}, Options{}); err == nil {
		t.Error("esperado erro sem argumentos")
	}
}

func TestRunDryRun(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{Default: "a", Projects: []config.Project{{Name: "a", Path: dir}}}
	if err := Run(cfg, Options{Command: "up", DryRun: true}); err != nil {
		t.Errorf("dry-run: %v", err)
	}
	cfg.Projects[0].Path = dir + "/inexistente"
	if err := Run(cfg, Options{Command: "up", DryRun: true}); err == nil {
		t.Error("esperado erro para diretório inexistente")
	}
}
