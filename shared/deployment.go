package shared

import (
	"fmt"
	"os"
	"strings"
	
	"gopkg.in/yaml.v3"
)

// DeploymentConfig represents the YAML deployment configuration
type DeploymentConfig struct {
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version"`
	Description string            `yaml:"description"`
	Env         map[string]string `yaml:"env,omitempty"`
	Jobs        yaml.Node         `yaml:"jobs"`
	jobOrder    []string          // Preserve order from YAML file
	jobMap      map[string]Job    // Parsed jobs
}

// Job represents a deployment job with multiple steps
type Job struct {
	Name  string            `yaml:"name"`
	Env   map[string]string `yaml:"env,omitempty"`
	Steps []Step            `yaml:"steps"`
}

// Step represents a single deployment step
type Step struct {
	Name string   `yaml:"name"`
	Cmd  string   `yaml:"cmd"`
	Args []string `yaml:"args,omitempty"`
}

// AllowedCommands lists the commands that are allowed in deployment steps
var AllowedCommands = map[string]bool{
	"echo": true,
	// Add more commands as needed:
	// "npm":    true,
	// "node":   true,
	// "docker": true,
	// "git":    true,
}

// ParseJobs parses the jobs from the YAML node and preserves order
func (config *DeploymentConfig) ParseJobs() error {
	config.jobMap = make(map[string]Job)
	config.jobOrder = []string{}
	
	if config.Jobs.Kind != yaml.MappingNode {
		return fmt.Errorf("jobs must be a mapping")
	}
	
	for i := 0; i < len(config.Jobs.Content); i += 2 {
		keyNode := config.Jobs.Content[i]
		valueNode := config.Jobs.Content[i+1]
		
		jobName := keyNode.Value
		config.jobOrder = append(config.jobOrder, jobName)
		
		var job Job
		if err := valueNode.Decode(&job); err != nil {
			return fmt.Errorf("failed to parse job '%s': %v", jobName, err)
		}
		
		config.jobMap[jobName] = job
	}
	
	return nil
}

// GetJob returns a job by name
func (config *DeploymentConfig) GetJob(name string) (Job, bool) {
	job, exists := config.jobMap[name]
	return job, exists
}

// ValidateConfig validates the deployment configuration
func (config *DeploymentConfig) ValidateConfig() error {
	if config.Name == "" {
		return fmt.Errorf("deployment name is required")
	}
	
	if err := config.ParseJobs(); err != nil {
		return fmt.Errorf("failed to parse jobs: %v", err)
	}
	
	if len(config.jobMap) == 0 {
		return fmt.Errorf("at least one job is required")
	}
	
	// Validate all jobs
	for jobName, job := range config.jobMap {
		if err := job.Validate(jobName); err != nil {
			return fmt.Errorf("job '%s': %v", jobName, err)
		}
	}
	
	return nil
}

// Validate validates a job
func (job *Job) Validate(jobName string) error {
	if job.Name == "" {
		return fmt.Errorf("job name is required")
	}
	
	if len(job.Steps) == 0 {
		return fmt.Errorf("at least one step is required")
	}
	
	// Validate all steps
	for i, step := range job.Steps {
		if err := step.Validate(); err != nil {
			return fmt.Errorf("step %d: %v", i+1, err)
		}
	}
	
	return nil
}

// Validate validates a single step
func (step *Step) Validate() error {
	if step.Name == "" {
		return fmt.Errorf("step name is required")
	}
	
	if step.Cmd == "" {
		return fmt.Errorf("step command is required")
	}
	
	if !AllowedCommands[step.Cmd] {
		return fmt.Errorf("command '%s' is not allowed. Allowed commands: %s", 
			step.Cmd, getAllowedCommandsList())
	}
	
	return nil
}

// GetJobOrder returns jobs in the order they appear in the YAML file
func (config *DeploymentConfig) GetJobOrder() []string {
	return config.jobOrder
}

// ParseYAMLConfig parses a YAML deployment configuration from a file
func ParseYAMLConfig(filename string) (*DeploymentConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}
	
	var config DeploymentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}
	
	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("validation failed: %v", err)
	}
	
	return &config, nil
}

func getAllowedCommandsList() string {
	var commands []string
	for cmd := range AllowedCommands {
		commands = append(commands, cmd)
	}
	return strings.Join(commands, ", ")
}