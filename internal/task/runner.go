package task

import (
	"fmt"
	"os"
	"strings"

	"docker-manager/internal/config"
	"docker-manager/internal/docker"
	"docker-manager/internal/ui"
)

type Options struct {
	Command string
	Args    []string
	Project string
	Profile string
	Full    bool
	Build   bool
	DryRun  bool
	Cwd     string
}

type step struct {
	args   []string
	detail string
}

type target struct {
	dir  string
	args []string
}

type definition struct {
	name        string
	summary     string
	usesProfile bool
	usesBuild   bool
	reverse     bool
	batch       func(config.Project, Options) step
	direct      func([]config.Project, Options) (target, error)
}

var definitions = []definition{
	{name: "up", summary: "sobe os serviços do perfil (basic por padrão) com docker compose up -d", usesProfile: true, usesBuild: true, batch: upStep},
	{name: "down", summary: "derruba os projetos com docker compose down, em ordem reversa", reverse: true, batch: downStep},
	{name: "stop", summary: "para os serviços do perfil, em ordem reversa", usesProfile: true, reverse: true, batch: stopStep},
	{name: "restart", summary: "reinicia os serviços do perfil", usesProfile: true, batch: restartStep},
	{name: "ps", summary: "lista os containers de cada projeto", batch: psStep},
	{name: "logs", summary: "acompanha os logs de um projeto: docker compose logs -f [argumentos]", direct: logsTarget},
	{name: "compose", summary: "executa docker compose <argumentos> no diretório de um projeto", direct: composeTarget},
	{name: "app", summary: "executa um comando no container app: docker exec -it <app> <argumentos>", direct: appTarget},
}

func lookup(name string) (definition, bool) {
	for _, def := range definitions {
		if def.name == name {
			return def, true
		}
	}
	return definition{}, false
}

func Exists(name string) bool {
	_, ok := lookup(name)
	return ok
}

func Passthrough(name string) bool {
	def, ok := lookup(name)
	return ok && def.direct != nil
}

func Names() []string {
	list := make([]string, 0, len(definitions))
	for _, def := range definitions {
		list = append(list, def.name)
	}
	return list
}

func Summaries() []string {
	list := make([]string, 0, len(definitions))
	for _, def := range definitions {
		list = append(list, fmt.Sprintf("%-8s %s", def.name, def.summary))
	}
	return list
}

type result struct {
	name   string
	detail string
	err    error
}

func Run(cfg *config.Config, opts Options) error {
	def, ok := lookup(opts.Command)
	if !ok {
		return fmt.Errorf("comando desconhecido: %q (disponíveis: %s)", opts.Command, strings.Join(Names(), ", "))
	}
	if err := validate(def, opts); err != nil {
		return err
	}
	if err := docker.Available(); err != nil {
		return err
	}
	selection, err := config.Select(cfg, opts.Project, opts.Cwd)
	if err != nil {
		return err
	}
	if def.direct != nil {
		return runDirect(def, selection.Projects, opts)
	}
	if opts.Profile != "" {
		if err := checkProfile(selection.Projects, opts.Profile); err != nil {
			return err
		}
	}
	return runBatch(def, selection, opts)
}

func validate(def definition, opts Options) error {
	if def.direct == nil && len(opts.Args) > 0 {
		return fmt.Errorf("argumento inesperado: %q", opts.Args[0])
	}
	if opts.Full && opts.Profile != "" {
		return fmt.Errorf("--full e --profile não podem ser usados juntos")
	}
	if opts.Full && !def.usesProfile {
		return fmt.Errorf("--full não se aplica ao comando %s", def.name)
	}
	if opts.Profile != "" && !def.usesProfile {
		return fmt.Errorf("--profile não se aplica ao comando %s", def.name)
	}
	if opts.Build && !def.usesBuild {
		return fmt.Errorf("--build não se aplica ao comando %s", def.name)
	}
	return nil
}

func checkProfile(projects []config.Project, profile string) error {
	for _, project := range projects {
		if _, ok := project.Profiles[profile]; ok {
			return nil
		}
	}
	return fmt.Errorf("perfil %q não existe em nenhum dos projetos selecionados (%s)", profile, joinNames(projects))
}

func runBatch(def definition, selection config.Selection, opts Options) error {
	projects := selection.Projects
	if def.reverse {
		projects = reversed(projects)
	}
	ui.Info("comando: %s | projetos: %s (%s)%s", def.name, joinNames(projects), selection.Origin, dryRunLabel(opts.DryRun))

	results := make([]result, 0, len(projects))
	for _, project := range projects {
		ui.Project(project.Name, project.Path)
		s := def.batch(project, opts)
		err := runStep(project, s, opts.DryRun)
		if err != nil {
			ui.Fail("%s: %v", project.Name, err)
		}
		results = append(results, result{name: project.Name, detail: s.detail, err: err})
	}
	return report(results)
}

func runStep(project config.Project, s step, dryRun bool) error {
	if info, err := os.Stat(project.Path); err != nil || !info.IsDir() {
		return fmt.Errorf("diretório inexistente: %s", project.Path)
	}
	args := docker.ComposeArgs(project.Files, s.args...)
	line := docker.Display(args)
	if dryRun {
		ui.Skipped(line)
		return nil
	}
	ui.Command(line)
	return docker.Run(project.Path, args)
}

func runDirect(def definition, projects []config.Project, opts Options) error {
	t, err := def.direct(projects, opts)
	if err != nil {
		return err
	}
	line := docker.Display(t.args)
	if opts.DryRun {
		ui.Skipped(line)
		return nil
	}
	ui.Trace(line)
	return docker.Exec(t.dir, t.args)
}

func report(results []result) error {
	ui.Section("Resumo")
	failures := 0
	for _, r := range results {
		if r.err != nil {
			failures++
			ui.StatusFail(r.name, r.err.Error())
			continue
		}
		ui.StatusOK(r.name, r.detail)
	}
	if failures > 0 {
		return fmt.Errorf("%d de %d projeto(s) falharam", failures, len(results))
	}
	return nil
}

func reversed(projects []config.Project) []config.Project {
	out := make([]config.Project, len(projects))
	for i, project := range projects {
		out[len(projects)-1-i] = project
	}
	return out
}

func joinNames(projects []config.Project) string {
	names := make([]string, 0, len(projects))
	for _, project := range projects {
		names = append(names, project.Name)
	}
	return strings.Join(names, ", ")
}

func dryRunLabel(dryRun bool) string {
	if dryRun {
		return " | modo dry-run (nenhum comando será executado)"
	}
	return ""
}
