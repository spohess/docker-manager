package docker

import (
	"reflect"
	"testing"
)

func TestComposeArgs(t *testing.T) {
	got := ComposeArgs([]string{"a.yml", "b.yml"}, "up", "-d")
	want := []string{"compose", "-f", "a.yml", "-f", "b.yml", "up", "-d"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("esperado %v, obtido %v", want, got)
	}
}

func TestExecArgs(t *testing.T) {
	if got := ExecArgs("app", true, []string{"php", "artisan"}); !reflect.DeepEqual(got, []string{"exec", "-it", "app", "php", "artisan"}) {
		t.Errorf("com tty: %v", got)
	}
	if got := ExecArgs("app", false, []string{"ls"}); !reflect.DeepEqual(got, []string{"exec", "-i", "app", "ls"}) {
		t.Errorf("sem tty: %v", got)
	}
}

func TestDisplay(t *testing.T) {
	got := Display([]string{"exec", "-it", "app", "php", "-r", "echo 1;"})
	if want := `docker exec -it app php -r "echo 1;"`; got != want {
		t.Errorf("esperado %s, obtido %s", want, got)
	}
}
