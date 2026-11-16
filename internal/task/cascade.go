package task

import (
	"fmt"
	"os"

	"docker-manager/internal/config"
	"docker-manager/internal/docker"
	"docker-manager/internal/ui"
)

func downDependents(project config.Project, dryRun bool) []result {
	if info, err := os.Stat(project.Path); err != nil || !info.IsDir() {
		return nil
	}
	dependents, err := docker.NetworkDependents(project.Path, project.Files)
	if err != nil {
		ui.Warn("não foi possível verificar os projetos conectados às redes de %s: %v", project.Name, err)
		return nil
	}
	results := make([]result, 0, len(dependents))
	for _, dependent := range dependents {
		ui.Warn("%s está conectado às redes de %s e será derrubado antes", dependent.Name, project.Name)
		args := docker.ComposeArgs(dependent.Files, "-p", dependent.Name, "down")
		err := execute(dependent.Dir, args, dryRun)
		if err != nil {
			ui.Fail("%s: %v", dependent.Name, err)
		}
		results = append(results, result{name: dependent.Name, detail: fmt.Sprintf("derrubado por usar a rede de %s", project.Name), err: err})
	}
	return results
}
