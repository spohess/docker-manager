package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"docker-manager/internal/cli"
	"docker-manager/internal/config"
	"docker-manager/internal/task"
	"docker-manager/internal/ui"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		ui.Fail("%v", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 1 && (args[0] == "-v" || args[0] == "--version") {
		args = []string{"version"}
	}
	inv, err := cli.Parse(args, task.Passthrough)
	if errors.Is(err, flag.ErrHelp) {
		usage(os.Stdout)
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w (veja docker-manager --help)", err)
	}

	switch inv.Command {
	case "", "help":
		usage(os.Stdout)
		return nil
	case "version":
		fmt.Printf("docker-manager %s\n", version)
		return nil
	}

	if !task.Exists(inv.Command) {
		return fmt.Errorf("comando desconhecido: %q (disponíveis: %s)", inv.Command, strings.Join(task.Names(), ", "))
	}

	cfg, path, err := config.Load(inv.Config)
	if err != nil {
		return err
	}
	if !task.Passthrough(inv.Command) {
		ui.Info("configuração: %s", path)
	}

	cwd, _ := os.Getwd()
	return task.Run(cfg, task.Options{
		Command: inv.Command,
		Args:    inv.Args,
		Project: inv.Project,
		Profile: inv.Profile,
		Full:    inv.Full,
		Build:   inv.Build,
		DryRun:  inv.DryRun,
		Cwd:     cwd,
	})
}

func usage(out *os.File) {
	fmt.Fprintf(out, `docker-manager %s - gerenciador docker compose para múltiplos projetos

Uso:
  dm <comando> [parâmetros] [argumentos]

Comandos:
`, version)
	for _, line := range task.Summaries() {
		fmt.Fprintf(out, "  %s\n", line)
	}
	fmt.Fprint(out, `
Parâmetros:
  -p, --project=nome   projeto(s) ou grupo(s) do config, separados por vírgula
  -f, --full           ignora o perfil e usa todos os serviços (up, stop, restart)
  --profile=nome       usa outro perfil em vez do basic (up, stop, restart)
  --build              acrescenta --build ao up
  --config=arquivo.yml caminho do arquivo de configuração
  --dry-run            mostra os comandos sem executar

Seleção de projetos:
  1. -p/--project, quando informado;
  2. o projeto cujo path contém o diretório atual;
  3. a chave default do config.

Nos comandos app, logs e compose, os parâmetros do dm vêm antes do comando ou
logo após ele; do primeiro argumento em diante tudo é repassado ao docker.
Use -- para repassar um argumento que começa com o nome de um parâmetro do dm.

Configuração:
  Sem --config o arquivo é procurado em DOCKER_MANAGER_CONFIG, em
  ./docker-manager.yml, em ~/.config/docker-manager/config.yml e em
  ~/.docker-manager.yml.

Exemplos:
  dm up                                 # default, perfil basic
  dm up -p backend -f                   # todos os serviços do backend
  dm up -p backend,admin --build
  dm up -p system                       # grupo: backend, admin e client
  dm down -p system                     # ordem reversa
  dm app php artisan migrate
  dm -p backend app php artisan tinker
  dm logs backend-app --tail=100
  dm compose pull
  dm compose -f docker-compose.x.yml up -d
`)
}
