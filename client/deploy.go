package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ten-james/go-deploy-system/shared/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func sendDeployment(serverURL string, zipData []byte, apiKey string, useYaml bool, verbose bool) error {
	serverAddr := strings.TrimPrefix(serverURL, "http://")
	serverAddr = strings.TrimPrefix(serverAddr, "https://")
	
	if !strings.Contains(serverAddr, ":") {
		serverAddr += ":9999"
	}

	// Set larger message size limits (100MB)
	maxMsgSize := 100 * 1024 * 1024
	conn, err := grpc.Dial(serverAddr, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMsgSize),
			grpc.MaxCallSendMsgSize(maxMsgSize),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := proto.NewDeploymentServiceClient(conn)

	// Increase timeout for large deployments (5 minutes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req := &proto.DeployRequest{
		ApiKey:         apiKey,
		DeploymentData: zipData,
		Filename:       "deployment.zip",
		UseYaml:        useYaml,
	}

	if useYaml {
		// Use streaming deployment for YAML
		return streamDeployment(client, ctx, req, verbose)
	} else {
		// Use traditional deployment for shell scripts
		resp, err := client.Deploy(ctx, req)
		if err != nil {
			return fmt.Errorf("deployment failed: %v", err)
		}

		if !resp.Success {
			return fmt.Errorf("deployment failed: %s", resp.Message)
		}

		if verbose {
			fmt.Printf("Deployment successful: %s\n", resp.Message)
			if resp.DeploymentId != "" {
				fmt.Printf("Deployment ID: %s\n", resp.DeploymentId)
			}
		}

		return nil
	}
}

func streamDeployment(client proto.DeploymentServiceClient, ctx context.Context, req *proto.DeployRequest, verbose bool) error {
	stream, err := client.DeployStream(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start deployment stream: %v", err)
	}
	
	for {
		log, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("stream error: %v", err)
		}

		// Format and display the log message
		displayExecutionLog(log, verbose)

		if log.Completed {
			break
		}
	}

	return nil
}

func displayExecutionLog(log *proto.ExecutionLog, verbose bool) {
	// Only show info logs when verbose is enabled
	if log.LogType == "info" && !verbose {
		return
	}
	// Format timestamp
	timestamp := log.Timestamp
	if len(timestamp) > 19 {
		timestamp = timestamp[:19] // Show only YYYY-MM-DD HH:MM:SS
	}

	// Color coding based on log type
	var icon, color string
	switch log.LogType {
	case "info":
		icon = "INFO"
		color = "\033[36m" // Cyan
	case "error":
		icon = "ERROR "
		color = "\033[31m" // Red
	case "stdout":
		icon = ""
		color = "\033[32m" // Green
	case "stderr":
		icon = "STDERR "
		color = "\033[33m" // Yellow
	default:
		icon = "LOG "
		color = "\033[37m" // White
	}

	reset := "\033[0m"

	// Format: [timestamp] job_name > step_name: message
	if log.JobName != "" && log.StepName != "" {
		fmt.Printf("%s[%s] %s%s > %s%s:\n %s\n\n", 
			icon, timestamp, color, log.JobName, log.StepName, reset, log.Message)
	} else if log.JobName != "" {
		fmt.Printf("%s[%s] %s%s%s:\n %s\n\n", 
			icon, timestamp, color, log.JobName, reset, log.Message)
	} else {
		fmt.Printf("%s[%s] %s\n", 
			icon, timestamp, log.Message)
	}
}