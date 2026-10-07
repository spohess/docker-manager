package task

import (
	"fmt"
	"strings"

	"docker-manager/internal/config"
)

func profileServices(project config.Project, opts Options) ([]string, string) {
	if opts.Full {
		return nil, "todos os serviços"
	}
	name := opts.Profile
	if name == "" {
		name = config.DefaultProfile
	}
	services, ok := project.Profiles[name]
	if !ok {
		return nil, fmt.Sprintf("todos os serviços (sem perfil %s)", name)
	}
	return services, fmt.Sprintf("perfil %s: %s", name, strings.Join(services, ", "))
}

func upStep(project config.Project, opts Options) step {
	services, label := profileServices(project, opts)
	args := []string{"up", "-d"}
	if opts.Build {
		args = append(args, "--build")
	}
	return step{args: append(args, services...), detail: label}
}

func downStep(config.Project, Options) step {
	return step{args: []string{"down"}, detail: "containers removidos"}
}

func stopStep(project config.Project, opts Options) step {
	services, label := profileServices(project, opts)
	return step{args: append([]string{"stop"}, services...), detail: label}
}

func restartStep(project config.Project, opts Options) step {
	services, label := profileServices(project, opts)
	return step{args: append([]string{"restart"}, services...), detail: label}
}

func psStep(config.Project, Options) step {
	return step{args: []string{"ps"}}
}
