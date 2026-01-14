package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/basphere/basphere-api/internal/config"
	"github.com/basphere/basphere-api/internal/handler"
	"github.com/basphere/basphere-api/internal/provisioner"
	"github.com/basphere/basphere-api/internal/store"
)

var (
	version = "0.1.0"
)

func main() {
	// Command line flags
	configPath := flag.String("config", "/etc/basphere/api.yaml", "Path to config file")
	host := flag.String("host", "", "Server host (overrides config)")
	port := flag.Int("port", 0, "Server port (overrides config)")
	showVersion := flag.Bool("version", false, "Show version")
	devMode := flag.Bool("dev", false, "Development mode (uses mock provisioner)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("basphere-api version %s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override with command line flags
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}

	// Initialize store
	fileStore, err := store.NewFileStore(cfg.Storage.PendingDir)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	// Initialize provisioner
	var prov provisioner.Provisioner
	if *devMode {
		log.Println("Running in development mode with mock provisioner")
		prov = provisioner.NewMockProvisioner()
	} else {
		bashProv, err := provisioner.NewBashProvisioner(cfg.Provisioner.AdminScript)
		if err != nil {
			log.Fatalf("Failed to initialize provisioner: %v", err)
		}
		prov = bashProv
	}

	// Find template directory
	templateDir := findTemplateDir()

	// Initialize handler
	h, err := handler.NewHandler(fileStore, prov, templateDir)
	if err != nil {
		log.Fatalf("Failed to initialize handler: %v", err)
	}

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting basphere-api server on %s", addr)
	log.Printf("Registration form: http://%s/register", addr)
	log.Printf("API endpoint: http://%s/api/v1", addr)

	if err := http.ListenAndServe(addr, h.Router()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// findTemplateDir finds the template directory
func findTemplateDir() string {
	// Check various locations
	candidates := []string{
		"/var/lib/basphere/api/templates",
		"/usr/local/share/basphere-api/templates",
		"./web/templates",
		"../web/templates",
	}

	// Also check relative to executable
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		candidates = append(candidates,
			filepath.Join(execDir, "templates"),
			filepath.Join(execDir, "../share/basphere-api/templates"),
		)
	}

	for _, dir := range candidates {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	// Default fallback
	return "./web/templates"
}
