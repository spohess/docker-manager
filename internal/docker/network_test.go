package docker

import (
	"reflect"
	"testing"
)

func TestParseNetworks(t *testing.T) {
	data := []byte(`{
		"name": "backend",
		"networks": {
			"appnet": {"name": "backend_appnet", "driver": "bridge"},
			"default": {"name": "backend_default"},
			"shared": {"name": "shared", "external": true}
		}
	}`)
	project, networks, err := parseNetworks(data)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if project != "backend" {
		t.Errorf("projeto esperado backend, obtido %s", project)
	}
	if want := []string{"backend_appnet", "backend_default"}; !reflect.DeepEqual(networks, want) {
		t.Errorf("esperado %v, obtido %v", want, networks)
	}
	if _, _, err := parseNetworks([]byte("x")); err == nil {
		t.Error("esperado erro para json inválido")
	}
}

func TestParseDependents(t *testing.T) {
	data := []byte("client\t/tmp/client\t/tmp/client/a.yml,/tmp/client/b.yml\n" +
		"backend\t/tmp/backend\t/tmp/backend/docker-compose.yml\n" +
		"client\t/tmp/client\t/tmp/client/a.yml,/tmp/client/b.yml\n" +
		"\t\t\n" +
		"admin\t/tmp/admin\t/tmp/admin/docker-compose.yml\n")
	got := parseDependents(data, "backend")
	want := []Dependent{
		{Name: "client", Dir: "/tmp/client", Files: []string{"/tmp/client/a.yml", "/tmp/client/b.yml"}},
		{Name: "admin", Dir: "/tmp/admin", Files: []string{"/tmp/admin/docker-compose.yml"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("esperado %v, obtido %v", want, got)
	}
}
