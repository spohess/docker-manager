package task

import (
	"fmt"

	"docker-manager/internal/config"
	"docker-manager/internal/docker"
)

func appTarget(projects []config.Project, opts Options) (target, error) {
	if len(opts.Args) == 0 {
		return target{}, fmt.Errorf("informe o comando a executar no container, ex.: dm app php artisan migrate")
	}
	project, err := appProject(projects)
	if err != nil {
		return target{}, err
	}
	return target{args: docker.ExecArgs(project.App, docker.Interactive(), opts.Args)}, nil
}

func appProject(projects []config.Project) (config.Project, error) {
	var candidates []config.Project
	for _, project := range projects {
		if project.App != "" {
			candidates = append(candidates, project)
		}
	}
	switch len(candidates) {
	case 1:
		return candidates[0], nil
	case 0:
		return config.Project{}, fmt.Errorf("nenhum dos projetos selecionados (%s) possui a chave \"app\"", joinNames(projects))
	}
	return config.Project{}, fmt.Errorf("mais de um projeto selecionado possui a chave \"app\" (%s); use -p para escolher um", joinNames(candidates))
}

func logsTarget(projects []config.Project, opts Options) (target, error) {
	project, err := singleProject("logs", projects)
	if err != nil {
		return target{}, err
	}
	args := append([]string{"logs", "-f"}, opts.Args...)
	return target{dir: project.Path, args: docker.ComposeArgs(project.Files, args...)}, nil
}

func composeTarget(projects []config.Project, opts Options) (target, error) {
	if len(opts.Args) == 0 {
		return target{}, fmt.Errorf("informe os argumentos do docker compose, ex.: dm compose pull")
	}
	project, err := singleProject("compose", projects)
	if err != nil {
		return target{}, err
	}
	return target{dir: project.Path, args: docker.ComposeArgs(project.Files, opts.Args...)}, nil
}

func singleProject(command string, projects []config.Project) (config.Project, error) {
	if len(projects) == 1 {
		return projects[0], nil
	}
	return config.Project{}, fmt.Errorf("o comando %s atua em um único projeto, mas %d foram selecionados (%s); use -p para escolher um", command, len(projects), joinNames(projects))
}
