package docker

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const dependentFormat = "{{.Label \"com.docker.compose.project\"}}\t{{.Label \"com.docker.compose.project.working_dir\"}}\t{{.Label \"com.docker.compose.project.config_files\"}}"

type Dependent struct {
	Name  string
	Dir   string
	Files []string
}

func NetworkDependents(dir string, files []string) ([]Dependent, error) {
	out, err := Output(dir, ComposeArgs(files, "config", "--format", "json"))
	if err != nil {
		return nil, err
	}
	project, networks, err := parseNetworks(out)
	if err != nil {
		return nil, err
	}
	if len(networks) == 0 {
		return nil, nil
	}
	args := []string{"ps", "-a", "--format", dependentFormat}
	for _, network := range networks {
		args = append(args, "--filter", "network="+network)
	}
	out, err = Output(dir, args)
	if err != nil {
		return nil, err
	}
	return parseDependents(out, project), nil
}

func parseNetworks(data []byte) (string, []string, error) {
	var cfg struct {
		Name     string `json:"name"`
		Networks map[string]struct {
			Name     string `json:"name"`
			External bool   `json:"external"`
		} `json:"networks"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", nil, fmt.Errorf("saída inválida do docker compose config: %w", err)
	}
	var networks []string
	for _, network := range cfg.Networks {
		if !network.External && network.Name != "" {
			networks = append(networks, network.Name)
		}
	}
	sort.Strings(networks)
	return cfg.Name, networks, nil
}

func parseDependents(data []byte, project string) []Dependent {
	var dependents []Dependent
	seen := make(map[string]bool)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			continue
		}
		name, dir, files := fields[0], fields[1], fields[2]
		if name == "" || name == project || seen[name] {
			continue
		}
		seen[name] = true
		dependents = append(dependents, Dependent{Name: name, Dir: dir, Files: splitFiles(files)})
	}
	return dependents
}

func splitFiles(raw string) []string {
	var files []string
	for _, file := range strings.Split(raw, ",") {
		if file = strings.TrimSpace(file); file != "" {
			files = append(files, file)
		}
	}
	return files
}
