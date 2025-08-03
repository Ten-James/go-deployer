package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ten-james/go-deploy-system/shared"
	"github.com/ten-james/go-deploy-system/shared/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type deploymentServer struct {
	proto.UnimplementedDeploymentServiceServer
}

func (s *deploymentServer) Deploy(ctx context.Context, req *proto.DeployRequest) (*proto.DeployResponse, error) {
	if req.ApiKey != apiKey {
		return nil, status.Errorf(codes.Unauthenticated, "Invalid API key")
	}

	if len(req.DeploymentData) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Empty deployment data")
	}

	timestamp := time.Now().Format("20060102-150405")
	deployDir := filepath.Join(uploadDir, fmt.Sprintf("deploy-%s", timestamp))
	
	if err := os.MkdirAll(deployDir, 0755); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create deployment directory: %v", err)
	}

	zipPath := filepath.Join(deployDir, "deployment.zip")
	if err := os.WriteFile(zipPath, req.DeploymentData, 0644); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to save deployment file: %v", err)
	}

	extractDir := filepath.Join(deployDir, "extracted")
	if err := unzip(zipPath, extractDir); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to extract deployment: %v", err)
	}

	// Handle legacy shell script deployment
	deployScript := filepath.Join(extractDir, "DEPLOY.sh")
	if _, err := os.Stat(deployScript); os.IsNotExist(err) {
		return nil, status.Errorf(codes.InvalidArgument, "DEPLOY.sh not found in deployment")
	}

	if err := os.Chmod(deployScript, 0755); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to make deploy script executable: %v", err)
	}

	go func() {
		defer cleanup(deployDir)
		
		cmd := exec.Command("/bin/bash", "DEPLOY.sh")
		cmd.Dir = extractDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			log.Printf("Deploy script failed: %v", err)
		} else {
			log.Printf("Deploy script completed successfully")
		}
	}()

	return &proto.DeployResponse{
		Success:      true,
		Message:      fmt.Sprintf("Deployment started successfully in %s", deployDir),
		DeploymentId: timestamp,
	}, nil
}

func (s *deploymentServer) DeployStream(req *proto.DeployRequest, stream proto.DeploymentService_DeployStreamServer) error {
	if req.ApiKey != apiKey {
		return status.Errorf(codes.Unauthenticated, "Invalid API key")
	}

	if len(req.DeploymentData) == 0 {
		return status.Errorf(codes.InvalidArgument, "Empty deployment data")
	}

	timestamp := time.Now().Format("20060102-150405")
	deployDir := filepath.Join(uploadDir, fmt.Sprintf("deploy-%s", timestamp))
	
	// Send initial log
	if err := sendLog(stream, "", "", "info", fmt.Sprintf("Starting deployment %s", timestamp)); err != nil {
		return err
	}

	if err := os.MkdirAll(deployDir, 0755); err != nil {
		return status.Errorf(codes.Internal, "Failed to create deployment directory: %v", err)
	}

	zipPath := filepath.Join(deployDir, "deployment.zip")
	if err := os.WriteFile(zipPath, req.DeploymentData, 0644); err != nil {
		return status.Errorf(codes.Internal, "Failed to save deployment file: %v", err)
	}

	extractDir := filepath.Join(deployDir, "extracted")
	if err := unzip(zipPath, extractDir); err != nil {
		return status.Errorf(codes.Internal, "Failed to extract deployment: %v", err)
	}

	if req.UseYaml {
		// Handle YAML deployment
		return s.executeYamlPipeline(extractDir, stream, deployDir)
	} else {
		// Handle shell script deployment with streaming
		return s.executeShellScript(extractDir, stream, deployDir)
	}
}

func (s *deploymentServer) GetDeploymentStatus(ctx context.Context, req *proto.StatusRequest) (*proto.StatusResponse, error) {
	if req.ApiKey != apiKey {
		return nil, status.Errorf(codes.Unauthenticated, "Invalid API key")
	}

	deployDir := filepath.Join(uploadDir, fmt.Sprintf("deploy-%s", req.DeploymentId))
	
	if _, err := os.Stat(deployDir); os.IsNotExist(err) {
		return &proto.StatusResponse{
			Status:    "not_found",
			Message:   "Deployment not found",
			Completed: true,
		}, nil
	}

	return &proto.StatusResponse{
		Status:    "running",
		Message:   "Deployment is in progress",
		Completed: false,
	}, nil
}

func sendLog(stream proto.DeploymentService_DeployStreamServer, jobName, stepName, logType, message string) error {
	log := &proto.ExecutionLog{
		Timestamp: time.Now().Format("15:04:05"),
		JobName:   jobName,
		StepName:  stepName,
		LogType:   logType,
		Message:   message,
		Completed: false,
	}
	return stream.Send(log)
}

func sendCompletionLog(stream proto.DeploymentService_DeployStreamServer, message string) error {
	log := &proto.ExecutionLog{
		Timestamp: time.Now().Format("15:04:05"),
		LogType:   "info",
		Message:   message,
		Completed: true,
	}
	return stream.Send(log)
}

func (s *deploymentServer) executeYamlPipeline(extractDir string, stream proto.DeploymentService_DeployStreamServer, deployDir string) error {
	defer cleanup(deployDir)
	
	// Look for deploy.yaml
	yamlPath := filepath.Join(extractDir, "deploy.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		sendLog(stream, "", "", "error", "deploy.yaml not found in deployment")
		return status.Errorf(codes.InvalidArgument, "deploy.yaml not found in deployment")
	}

	// Parse YAML configuration
	config, err := shared.ParseYAMLConfig(yamlPath)
	if err != nil {
		sendLog(stream, "", "", "error", fmt.Sprintf("Failed to parse YAML: %v", err))
		return status.Errorf(codes.InvalidArgument, "Failed to parse YAML: %v", err)
	}

	sendLog(stream, "", "", "info", fmt.Sprintf("Starting pipeline: %s", config.Name))
	
	// Execute jobs in order
	jobOrder := config.GetJobOrder()
	for _, jobName := range jobOrder {
		job, exists := config.GetJob(jobName)
		if !exists {
			sendLog(stream, jobName, "", "error", "Job not found")
			continue
		}

		sendLog(stream, jobName, "", "info", fmt.Sprintf("Starting job: %s", job.Name))
		
		// Prepare environment variables
		env := buildEnvironment(config.Env, job.Env)
		
		// Execute each step in the job
		for _, step := range job.Steps {
			if err := s.executeStep(extractDir, jobName, step, env, stream); err != nil {
				sendLog(stream, jobName, step.Name, "error", fmt.Sprintf("Step failed: %v", err))
				return err
			}
		}
		
		sendLog(stream, jobName, "", "info", fmt.Sprintf("Job completed: %s", job.Name))
	}
	
	sendCompletionLog(stream, "Pipeline completed successfully")
	return nil
}

func (s *deploymentServer) executeShellScript(extractDir string, stream proto.DeploymentService_DeployStreamServer, deployDir string) error {
	defer cleanup(deployDir)
	
	deployScript := filepath.Join(extractDir, "DEPLOY.sh")
	if _, err := os.Stat(deployScript); os.IsNotExist(err) {
		sendLog(stream, "", "", "error", "DEPLOY.sh not found in deployment")
		return status.Errorf(codes.InvalidArgument, "DEPLOY.sh not found in deployment")
	}

	if err := os.Chmod(deployScript, 0755); err != nil {
		sendLog(stream, "", "", "error", fmt.Sprintf("Failed to make script executable: %v", err))
		return status.Errorf(codes.Internal, "Failed to make deploy script executable: %v", err)
	}

	sendLog(stream, "", "", "info", "Executing DEPLOY.sh script")
	
	cmd := exec.Command("/bin/bash", "DEPLOY.sh")
	cmd.Dir = extractDir
	
	// Capture output and stream it
	output, err := cmd.CombinedOutput()
	if err != nil {
		sendLog(stream, "", "", "error", fmt.Sprintf("Script failed: %v", err))
		sendLog(stream, "", "", "stderr", string(output))
		return status.Errorf(codes.Internal, "Deploy script failed: %v", err)
	}
	
	if len(output) > 0 {
		sendLog(stream, "", "", "stdout", string(output))
	}
	
	sendCompletionLog(stream, "Shell script completed successfully")
	return nil
}

func (s *deploymentServer) executeStep(workDir, jobName string, step shared.Step, env []string, stream proto.DeploymentService_DeployStreamServer) error {
	sendLog(stream, jobName, step.Name, "info", fmt.Sprintf("Executing: %s %s", step.Cmd, strings.Join(step.Args, " ")))

	log.Println("Executing step:", step.Name, "Command:", step.Cmd, "Args:", step.Args)
	
	// Handle echo command interpretation in Go instead of executing it
	if step.Cmd == "echo" {
		output := s.interpretEcho(step.Args, env)
		sendLog(stream, jobName, step.Name, "stdout", output)
		sendLog(stream, jobName, step.Name, "info", "Step completed successfully")
		return nil
	}
	
	// Create command for other commands
	cmd := exec.Command(step.Cmd, step.Args...)
	cmd.Dir = workDir
	cmd.Env = env
	
	// Execute command and capture output
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		sendLog(stream, jobName, step.Name, "error", fmt.Sprintf("Command failed: %v", err))
		if len(output) > 0 {
			sendLog(stream, jobName, step.Name, "stderr", string(output))
		}
		return err
	}
	
	if len(output) > 0 {
		sendLog(stream, jobName, step.Name, "stdout", strings.TrimSpace(string(output)))
	}
	
	sendLog(stream, jobName, step.Name, "info", "Step completed successfully")
	return nil
}

func (s *deploymentServer) interpretEcho(args []string, env []string) string {
	// Create a map for environment variables for easy lookup
	envMap := make(map[string]string)
	for _, envVar := range env {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	
	if len(args) == 0 {
		return ""
	}
	
	// Join all arguments with spaces (like real echo does)
	message := strings.Join(args, " ")
	
	// Handle basic variable expansion (like $VAR or ${VAR})
	message = os.Expand(message, func(key string) string {
		if value, exists := envMap[key]; exists {
			return value
		}
		return os.Getenv(key) // Fallback to system environment
	})
	
	return message
}

func buildEnvironment(globalEnv, jobEnv map[string]string) []string {
	// Start with system environment
	env := os.Environ()
	
	// Add global environment variables
	for key, value := range globalEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	
	// Add job-specific environment variables (override global if same key)
	for key, value := range jobEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	
	return env
}