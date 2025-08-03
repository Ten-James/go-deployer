package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ten-james/go-deploy-system/shared"
)

var verbose bool

func verbosePrintf(format string, args ...interface{}) {
	if verbose {
		fmt.Printf(format, args...)
	}
}

func verbosePrintln(args ...interface{}) {
	if verbose {
		fmt.Println(args...)
	}
}

func main() {
	var skipConfirmation bool
	flag.BoolVar(&skipConfirmation, "y", false, "Skip deployment confirmation prompt")
	flag.BoolVar(&skipConfirmation, "yes", false, "Skip deployment confirmation prompt")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	
	if len(os.Args) > 1 {
		if os.Args[1] == "help" {
			fmt.Println("Usage: go-deploy [options] <server-url|config>")
			fmt.Println("Commands:")
			fmt.Println("  go-deploy config <api-key> <server-url>     - Set API key and server URL")
			fmt.Println("  go-deploy [options] <server-url>            - Deploy to specified server")
			fmt.Println("  go-deploy [options]                         - Deploy using config server URL")
			fmt.Println("Options:")
			fmt.Println("  -y, --yes                                   - Skip deployment confirmation prompt")
			fmt.Println("  -v, --verbose                               - Enable verbose output")
			os.Exit(1)
		}

		if os.Args[1] == "config" {
			if len(os.Args) < 4 {
				fmt.Println("Usage: go-deploy config <api-key> <server-url>")
				os.Exit(1)
			}
			if err := saveConfig(os.Args[2], os.Args[3]); err != nil {
				fmt.Printf("Failed to save config: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Configuration saved successfully")
			return
		}
	}
	
	flag.Parse()

	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		fmt.Println("Run 'go-deploy config <api-key> <server-url>' to set your configuration")
		os.Exit(1)
	}

	var serverURL string
	args := flag.Args()
	if len(args) > 0 {
		serverURL = args[0]
	} else {
		if config.ServerURL == "" {
			fmt.Println("No server URL provided. Use 'go-deploy <server-url>' or set it in config")
			os.Exit(1)
		}
		serverURL = config.ServerURL
	}

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	deployYaml := filepath.Join(currentDir, "deploy.yaml")
	deployScript := filepath.Join(currentDir, "DEPLOY.sh")
	
	yamlExists := fileExists(deployYaml)
	scriptExists := fileExists(deployScript)
	
	var deploymentFile string
	var useYaml bool
	
	if yamlExists && scriptExists {
		// Both exist - warn and use YAML
		verbosePrintf("\n⚠️  WARNING: Both deploy.yaml and DEPLOY.sh found!\n")
		verbosePrintf("   Using deploy.yaml (recommended format)\n")
		verbosePrintf("   Consider removing DEPLOY.sh to avoid confusion\n\n")
		deploymentFile = deployYaml
		useYaml = true
	} else if yamlExists {
		// Only YAML exists - use it
		deploymentFile = deployYaml
		useYaml = true
	} else if scriptExists {
		// Only script exists - warn about migration
		verbosePrintf("\n⚠️  MIGRATION NOTICE: Found DEPLOY.sh (legacy format)\n")
		verbosePrintf("   Consider migrating to deploy.yaml for better CI/CD pipeline support\n")
		verbosePrintf("   YAML format supports multiple jobs, environment variables, and better structure\n\n")
		deploymentFile = deployScript
		useYaml = false
	} else {
		// Neither exists
		fmt.Println("No deployment configuration found!")
		fmt.Println("Please create either:")
		fmt.Println("  - deploy.yaml (recommended - CI/CD pipeline format)")
		fmt.Println("  - DEPLOY.sh (legacy shell script format)")
		os.Exit(1)
	}

	// Validate YAML configuration if using YAML format
	if useYaml {
		if err := validateYamlConfig(deploymentFile); err != nil {
			fmt.Printf("❌ YAML validation failed: %v\n", err)
			os.Exit(1)
		}
		verbosePrintf("✅ YAML configuration is valid\n\n")
	}

	// Confirm deployment based on file type
	if !skipConfirmation && !confirmDeployment(deploymentFile, useYaml) {
		fmt.Println("Deployment cancelled by user")
		os.Exit(0)
	}

	verbosePrintln("Packaging deployment...")
	zipData, err := createZip(currentDir)
	if err != nil {
		fmt.Printf("Failed to create deployment package: %v\n", err)
		os.Exit(1)
	}

	if err := sendDeployment(serverURL, zipData, config.APIKey, useYaml, verbose); err != nil {
		fmt.Printf("Failed to send deployment: %v\n", err)
		os.Exit(1)
	}

}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func validateYamlConfig(filename string) error {
	_, err := shared.ParseYAMLConfig(filename)
	return err
}

func confirmDeployment(deployFile string, useYaml bool) bool {
	if useYaml {
		verbosePrintf("\n⚠️  WARNING: deploy.yaml found in current directory!\n")
		verbosePrintf("   Configuration: %s\n", deployFile)
		verbosePrintf("   This pipeline will be executed on the remote server during deployment.\n")
		verbosePrintf("   Please review the configuration before proceeding.\n\n")
		
		// Show YAML preview for quick review
		if verbose {
			if content, err := os.ReadFile(deployFile); err == nil {
				lines := strings.Split(string(content), "\n")
				fmt.Printf("Configuration preview (first 15 lines):\n")
				fmt.Printf("----------------------------------\n")
				for i, line := range lines {
					if i >= 15 {
						if len(lines) > 15 {
							fmt.Printf("... (%d more lines)\n", len(lines)-15)
						}
						break
					}
					fmt.Printf("%2d: %s\n", i+1, line)
				}
				fmt.Printf("----------------------------------\n\n")
			}
		}
	} else {
		verbosePrintf("\n⚠️  WARNING: DEPLOY.sh found in current directory!\n")
		verbosePrintf("   Script: %s\n", deployFile)
		verbosePrintf("   This script will be executed on the remote server during deployment.\n")
		verbosePrintf("   Please review the script contents before proceeding.\n\n")
		
		// Show first few lines of the script for quick review
		if verbose {
			if content, err := os.ReadFile(deployFile); err == nil {
				lines := strings.Split(string(content), "\n")
				fmt.Printf("Script preview (first 10 lines):\n")
				fmt.Printf("----------------------------------\n")
				for i, line := range lines {
					if i >= 10 {
						if len(lines) > 10 {
							fmt.Printf("... (%d more lines)\n", len(lines)-10)
						}
						break
					}
					fmt.Printf("%2d: %s\n", i+1, line)
				}
				fmt.Printf("----------------------------------\n\n")
			}
		}
	}
	
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Do you want to proceed with deployment? [y/N]: ")
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return false
		}
		
		response = strings.ToLower(strings.TrimSpace(response))
		switch response {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		default:
			fmt.Println("Please answer 'y' (yes) or 'n' (no)")
		}
	}
}