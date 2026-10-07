package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultProfile = "basic"
	envPath        = "DOCKER_MANAGER_CONFIG"
)

const (
	OriginFlag    = "--project"
	OriginCwd     = "diretório atual"
	OriginDefault = "default"
)

type Project struct {
	Name     string              `yaml:"name"`
	Path     string              `yaml:"path"`
	App      string              `yaml:"app"`
	Files    []string            `yaml:"files"`
	Profiles map[string][]string `yaml:"profiles"`
}

type Config struct {
	Default  string              `yaml:"default"`
	Projects []Project           `yaml:"projects"`
	Groups   map[string][]string `yaml:"groups"`
}

type Selection struct {
	Projects []Project
	Origin   string
}

func candidatePaths() []string {
	paths := []string{
		"docker-manager.yml",
		"docker-manager.yaml",
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".config", "docker-manager", "config.yml"),
			filepath.Join(home, ".config", "docker-manager", "config.yaml"),
			filepath.Join(home, ".docker-manager.yml"),
			filepath.Join(home, ".docker-manager.yaml"),
		)
	}
	return paths
}

func Load(path string) (*Config, string, error) {
	resolved, err := resolvePath(path)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, "", fmt.Errorf("não foi possível ler %s: %w", resolved, err)
	}
	cfg, err := Parse(data)
	if err != nil {
		return nil, resolved, fmt.Errorf("%s: %w", resolved, err)
	}
	return cfg, resolved, nil
}

func Parse(data []byte) (*Config, error) {
	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("yaml inválido: %w", err)
	}
	if len(cfg.Projects) == 0 {
		return nil, fmt.Errorf("nenhum projeto declarado na chave \"projects\"")
	}
	seen := make(map[string]bool, len(cfg.Projects))
	for i := range cfg.Projects {
		p := &cfg.Projects[i]
		if err := normalizeProject(p, i); err != nil {
			return nil, err
		}
		key := strings.ToLower(p.Name)
		if seen[key] {
			return nil, fmt.Errorf("projeto %q está duplicado", p.Name)
		}
		seen[key] = true
	}
	if err := normalizeGroups(&cfg); err != nil {
		return nil, err
	}
	cfg.Default = strings.TrimSpace(cfg.Default)
	if cfg.Default != "" {
		if _, err := cfg.resolve(cfg.Default); err != nil {
			return nil, fmt.Errorf("chave \"default\": %w", err)
		}
	}
	return &cfg, nil
}

func normalizeProject(p *Project, index int) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return fmt.Errorf("projeto #%d: chave \"name\" é obrigatória", index+1)
	}
	if strings.Contains(p.Name, ",") {
		return fmt.Errorf("projeto %q: o nome não pode conter vírgula", p.Name)
	}
	if strings.TrimSpace(p.Path) == "" {
		return fmt.Errorf("projeto %q: chave \"path\" é obrigatória", p.Name)
	}
	path, err := expand(p.Path)
	if err != nil {
		return fmt.Errorf("projeto %q: %w", p.Name, err)
	}
	p.Path = path
	p.App = strings.TrimSpace(p.App)
	for i, file := range p.Files {
		file = strings.TrimSpace(file)
		if file == "" {
			return fmt.Errorf("projeto %q: item #%d de \"files\" está vazio", p.Name, i+1)
		}
		if p.Files[i], err = expandFile(file); err != nil {
			return fmt.Errorf("projeto %q: %w", p.Name, err)
		}
	}
	profiles, err := normalizeProfiles(p.Profiles)
	if err != nil {
		return fmt.Errorf("projeto %q: %w", p.Name, err)
	}
	p.Profiles = profiles
	return nil
}

func normalizeProfiles(raw map[string][]string) (map[string][]string, error) {
	profiles := make(map[string][]string, len(raw))
	for name, services := range raw {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("perfil sem nome em \"profiles\"")
		}
		cleaned, err := cleanList(services)
		if err != nil {
			return nil, fmt.Errorf("perfil %q: %w", name, err)
		}
		profiles[name] = cleaned
	}
	return profiles, nil
}

func normalizeGroups(cfg *Config) error {
	groups := make(map[string][]string, len(cfg.Groups))
	seen := make(map[string]bool, len(cfg.Groups))
	for name, members := range cfg.Groups {
		name = strings.TrimSpace(name)
		key := strings.ToLower(name)
		switch {
		case name == "":
			return fmt.Errorf("grupo sem nome em \"groups\"")
		case strings.Contains(name, ","):
			return fmt.Errorf("grupo %q: o nome não pode conter vírgula", name)
		case seen[key]:
			return fmt.Errorf("grupo %q está duplicado", name)
		}
		if _, ok := cfg.find(name); ok {
			return fmt.Errorf("grupo %q tem o mesmo nome de um projeto", name)
		}
		cleaned, err := cleanList(members)
		if err != nil {
			return fmt.Errorf("grupo %q: %w", name, err)
		}
		for i, member := range cleaned {
			project, ok := cfg.find(member)
			if !ok {
				return fmt.Errorf("grupo %q: projeto %q não encontrado", name, member)
			}
			cleaned[i] = project.Name
		}
		seen[key] = true
		groups[name] = cleaned
	}
	cfg.Groups = groups
	return nil
}

func cleanList(items []string) ([]string, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("lista vazia")
	}
	cleaned := make([]string, 0, len(items))
	for i, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			return nil, fmt.Errorf("item #%d está vazio", i+1)
		}
		cleaned = append(cleaned, item)
	}
	return cleaned, nil
}

func Select(cfg *Config, raw, cwd string) (Selection, error) {
	if strings.TrimSpace(raw) != "" {
		projects, err := cfg.resolve(raw)
		return Selection{Projects: projects, Origin: OriginFlag}, err
	}
	if project, ok := cfg.projectAt(cwd); ok {
		return Selection{Projects: []Project{project}, Origin: OriginCwd}, nil
	}
	if cfg.Default != "" {
		projects, err := cfg.resolve(cfg.Default)
		return Selection{Projects: projects, Origin: OriginDefault}, err
	}
	return Selection{}, fmt.Errorf("nenhum projeto selecionado; informe -p/--project=nome, execute dentro do diretório de um projeto ou defina \"default\" na configuração")
}

func (cfg *Config) resolve(raw string) ([]Project, error) {
	var selected []Project
	seen := make(map[string]bool)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		members, err := cfg.expandName(part)
		if err != nil {
			return nil, err
		}
		for _, member := range members {
			project, _ := cfg.find(member)
			key := strings.ToLower(project.Name)
			if seen[key] {
				continue
			}
			seen[key] = true
			selected = append(selected, project)
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("nenhum projeto informado")
	}
	return selected, nil
}

func (cfg *Config) expandName(name string) ([]string, error) {
	if project, ok := cfg.find(name); ok {
		return []string{project.Name}, nil
	}
	for group, members := range cfg.Groups {
		if strings.EqualFold(group, name) {
			return members, nil
		}
	}
	return nil, fmt.Errorf("projeto ou grupo %q não encontrado na configuração (disponíveis: %s)", name, cfg.available())
}

func (cfg *Config) find(name string) (Project, bool) {
	for _, p := range cfg.Projects {
		if strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return Project{}, false
}

func (cfg *Config) available() string {
	names := make([]string, 0, len(cfg.Projects)+len(cfg.Groups))
	for _, p := range cfg.Projects {
		names = append(names, p.Name)
	}
	groups := make([]string, 0, len(cfg.Groups))
	for group := range cfg.Groups {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	return strings.Join(append(names, groups...), ", ")
}

func (cfg *Config) projectAt(dir string) (Project, bool) {
	if strings.TrimSpace(dir) == "" {
		return Project{}, false
	}
	dir = realPath(dir)
	var best Project
	bestLen := -1
	for _, p := range cfg.Projects {
		root := realPath(p.Path)
		if within(root, dir) && len(root) > bestLen {
			best, bestLen = p, len(root)
		}
	}
	return best, bestLen >= 0
}

func within(root, dir string) bool {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func realPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
}

func resolvePath(path string) (string, error) {
	if strings.TrimSpace(path) != "" {
		expanded, err := expand(path)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(expanded); err != nil {
			return "", fmt.Errorf("arquivo de configuração não encontrado: %s", expanded)
		}
		return expanded, nil
	}
	if fromEnv := strings.TrimSpace(os.Getenv(envPath)); fromEnv != "" {
		return resolvePath(fromEnv)
	}
	for _, candidate := range candidatePaths() {
		expanded, err := expand(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(expanded); err == nil && !info.IsDir() {
			return expanded, nil
		}
	}
	return "", fmt.Errorf("arquivo de configuração não encontrado; informe --config=caminho.yml, defina %s ou crie um dos arquivos: %s", envPath, strings.Join(candidatePaths(), ", "))
}

func expandFile(file string) (string, error) {
	if strings.HasPrefix(file, "~") || filepath.IsAbs(file) {
		return expand(file)
	}
	return os.ExpandEnv(file), nil
}

func expand(path string) (string, error) {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("não foi possível resolver \"~\" em %s: %w", path, err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	path = os.ExpandEnv(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("caminho inválido %s: %w", path, err)
	}
	return abs, nil
}
