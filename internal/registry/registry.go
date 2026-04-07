package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the global user preferences.
type Config struct {
	DefaultTarget string `yaml:"default_target,omitempty"`
}

// Project represents a single registered xcaffold project.
type Project struct {
	Path        string    `yaml:"path"`
	Name        string    `yaml:"name"`
	Registered  time.Time `yaml:"registered"`
	Targets     []string  `yaml:"targets,omitempty"`
	LastApplied time.Time `yaml:"last_applied,omitempty"`
}

// GlobalHome returns the absolute path to ~/.xcaffold/.
func GlobalHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".xcaffold"), nil
}

// EnsureGlobalHome ensures ~/.xcaffold/ exists along with empty configuration files.
func EnsureGlobalHome() error {
	home, err := GlobalHome()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(home, 0755); err != nil {
		return fmt.Errorf("could not create global home: %w", err)
	}

	configPath := filepath.Join(home, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := Config{DefaultTarget: "claude"}
		out, _ := yaml.Marshal(cfg)
		os.WriteFile(configPath, out, 0600)
	}

	projectsPath := filepath.Join(home, "projects.yaml")
	if _, err := os.Stat(projectsPath); os.IsNotExist(err) {
		out, _ := yaml.Marshal([]Project{})
		os.WriteFile(projectsPath, out, 0600)
	}

	return nil
}

func readProjects() ([]Project, error) {
	home, err := GlobalHome()
	if err != nil {
		return nil, err
	}
	projectsPath := filepath.Join(home, "projects.yaml")
	data, err := os.ReadFile(projectsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Project{}, nil
		}
		return nil, err
	}
	var projects []Project
	if err := yaml.Unmarshal(data, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func writeProjects(projects []Project) error {
	home, err := GlobalHome()
	if err != nil {
		return err
	}
	projectsPath := filepath.Join(home, "projects.yaml")
	data, err := yaml.Marshal(projects)
	if err != nil {
		return err
	}
	return os.WriteFile(projectsPath, data, 0600)
}

// Register adds a new project or updates an existing one by path.
func Register(projectPath, name string, targets []string) error {
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return err
	}

	projects, err := readProjects()
	if err != nil {
		return err
	}

	found := false
	for i, p := range projects {
		if p.Path == abs {
			// Found by path, update it. Focus on name, targets.
			projects[i].Name = name
			if len(targets) > 0 {
				projects[i].Targets = targets
			}
			found = true
			break
		}
	}

	if !found {
		// New project, but check name collision.
		for _, p := range projects {
			if p.Name == name {
				// Name collision! Use unique suffix (the parent dir)
				name = fmt.Sprintf("%s-%s", filepath.Base(filepath.Dir(abs)), name)
				break
			}
		}
		projects = append(projects, Project{
			Path:       abs,
			Name:       name,
			Registered: time.Now().UTC(),
			Targets:    targets,
		})
	}

	return writeProjects(projects)
}

// Unregister removes a project by name or path.
func Unregister(nameOrPath string) error {
	projects, err := readProjects()
	if err != nil {
		return err
	}
	
	abs, _ := filepath.Abs(nameOrPath) // best effort for path match
	var filtered []Project
	for _, p := range projects {
		if p.Name != nameOrPath && p.Path != abs {
			filtered = append(filtered, p)
		}
	}

	return writeProjects(filtered)
}

// List returns all registered projects.
func List() ([]Project, error) {
	return readProjects()
}

// Resolve looks up a project by its registered name or absolute path.
func Resolve(nameOrPath string) (Project, error) {
	projects, err := readProjects()
	if err != nil {
		return Project{}, err
	}

	abs, _ := filepath.Abs(nameOrPath)

	for _, p := range projects {
		if p.Name == nameOrPath || p.Path == abs || p.Path == nameOrPath {
			return p, nil
		}
	}

	return Project{}, fmt.Errorf("project not found: %s", nameOrPath)
}

// UpdateLastApplied updates the LastApplied timestamp for a project.
func UpdateLastApplied(projectPath string) error {
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return err
	}

	projects, err := readProjects()
	if err != nil {
		return err
	}

	for i, p := range projects {
		if p.Path == abs {
			projects[i].LastApplied = time.Now().UTC()
			return writeProjects(projects)
		}
	}
	return nil
}
