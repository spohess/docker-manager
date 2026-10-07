package cli

import (
	"errors"
	"flag"
	"reflect"
	"strings"
	"testing"
)

func passthrough(name string) bool {
	return name == "app" || name == "logs" || name == "compose"
}

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Invocation
	}{
		{name: "somente comando", args: []string{"up"}, want: Invocation{Command: "up"}},
		{name: "flags depois do comando", args: []string{"up", "-p", "a,b", "-f"}, want: Invocation{Command: "up", Options: Options{Project: "a,b", Full: true}}},
		{name: "flags antes do comando", args: []string{"-p", "x", "up", "--full"}, want: Invocation{Command: "up", Options: Options{Project: "x", Full: true}}},
		{name: "formato com igual", args: []string{"up", "--project=x", "--build", "--dry-run", "--profile=queue"}, want: Invocation{Command: "up", Options: Options{Project: "x", Build: true, DryRun: true, Profile: "queue"}}},
		{name: "app com projeto antes", args: []string{"-p", "proj", "app", "php", "artisan", "migrate", "--force"}, want: Invocation{Command: "app", Args: []string{"php", "artisan", "migrate", "--force"}, Options: Options{Project: "proj"}}},
		{name: "app com projeto depois", args: []string{"app", "-p", "proj", "php", "artisan", "test", "-p"}, want: Invocation{Command: "app", Args: []string{"php", "artisan", "test", "-p"}, Options: Options{Project: "proj"}}},
		{name: "compose repassa -f", args: []string{"compose", "-f", "docker-compose.x.yml", "up"}, want: Invocation{Command: "compose", Args: []string{"-f", "docker-compose.x.yml", "up"}}},
		{name: "separador", args: []string{"app", "--", "--dry-run"}, want: Invocation{Command: "app", Args: []string{"--dry-run"}}},
		{name: "logs repassa flags desconhecidas", args: []string{"logs", "--tail=50", "app"}, want: Invocation{Command: "logs", Args: []string{"--tail=50", "app"}}},
		{name: "full antes do app", args: []string{"-f", "app", "ls"}, want: Invocation{Command: "app", Args: []string{"ls"}, Options: Options{Full: true}}},
		{name: "posicional extra", args: []string{"up", "extra"}, want: Invocation{Command: "up", Args: []string{"extra"}}},
		{name: "dry-run depois do app", args: []string{"app", "--dry-run", "ls", "-la"}, want: Invocation{Command: "app", Args: []string{"ls", "-la"}, Options: Options{DryRun: true}}},
	}
	for _, tc := range cases {
		got, err := Parse(tc.args, passthrough)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: esperado %+v, obtido %+v", tc.name, tc.want, got)
		}
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse([]string{"-h"}, passthrough); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("-h deveria retornar flag.ErrHelp, obtido %v", err)
	}
	if _, err := Parse([]string{"up", "--help"}, passthrough); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("--help deveria retornar flag.ErrHelp, obtido %v", err)
	}
	if _, err := Parse([]string{"up", "--xyz"}, passthrough); err == nil || !strings.Contains(err.Error(), "xyz") {
		t.Errorf("esperado erro de flag desconhecida, obtido %v", err)
	}
	if _, err := Parse([]string{"up", "-p"}, passthrough); err == nil {
		t.Error("esperado erro para -p sem valor")
	}
}
