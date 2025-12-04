package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Port         string
	SecretKey    string
	RepoPath     string
	Branch       string
	DeployScript string
	AllowedIPs   []string
}

type WebhookPayload struct {
	Ref        string `json:"ref"`
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	HeadCommit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"head_commit"`
}

type Deployer struct {
	config    *Config
	mutex     sync.Mutex
	deploying bool
}

func main() {
	var (
		port         = flag.String("port", "9000", "Port to listen on")
		secretKey    = flag.String("secret", "", "GitHub webhook secret key (or set WEBHOOK_SECRET env var)")
		repoPath     = flag.String("repo", "/opt/lightning-node-tools", "Path to repository on server")
		branch       = flag.String("branch", "main", "Branch to deploy")
		deployScript = flag.String("script", "./scripts/auto-deploy.sh", "Deployment script to run")
	)
	flag.Parse()

	// Get secret from environment if not provided
	if *secretKey == "" {
		*secretKey = os.Getenv("WEBHOOK_SECRET")
	}
	if *secretKey == "" {
		log.Fatal("❌ Webhook secret required! Use --secret flag or WEBHOOK_SECRET environment variable")
	}

	config := &Config{
		Port:         *port,
		SecretKey:    *secretKey,
		RepoPath:     *repoPath,
		Branch:       *branch,
		DeployScript: *deployScript,
		// Add your server IPs here for additional security
		AllowedIPs: []string{}, // Empty means allow all (GitHub webhooks come from various IPs)
	}

	deployer := &Deployer{config: config}

	http.HandleFunc("/webhook", deployer.handleWebhook)
	http.HandleFunc("/health", deployer.handleHealth)
	http.HandleFunc("/status", deployer.handleStatus)

	log.Printf("🚀 Webhook deployer starting on port %s", config.Port)
	log.Printf("📁 Repository path: %s", config.RepoPath)
	log.Printf("🌿 Target branch: %s", config.Branch)
	log.Printf("📜 Deploy script: %s", config.DeployScript)

	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}

func (d *Deployer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify GitHub signature
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		log.Printf("❌ Missing signature header")
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Error reading request body: %v", err)
		http.Error(w, "Error reading body", http.StatusBadRequest)
		return
	}

	if !d.verifySignature(signature, body) {
		log.Printf("❌ Invalid signature from %s", r.RemoteAddr)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse webhook payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("❌ Error parsing webhook payload: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Check if this is a push to our target branch
	expectedRef := "refs/heads/" + d.config.Branch
	if payload.Ref != expectedRef {
		log.Printf("ℹ️  Ignoring push to branch %s (waiting for %s)", payload.Ref, expectedRef)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Branch ignored"))
		return
	}

	log.Printf("🎯 Webhook received for %s", payload.Repository.FullName)
	log.Printf("📝 Commit: %s", payload.HeadCommit.ID[:8])
	log.Printf("💬 Message: %s", payload.HeadCommit.Message)
	log.Printf("👤 Author: %s <%s>", payload.HeadCommit.Author.Name, payload.HeadCommit.Author.Email)

	// Start deployment in background
	go d.deploy(payload)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Deployment triggered"))
}

func (d *Deployer) verifySignature(signature string, body []byte) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	hash := hmac.New(sha256.New, []byte(d.config.SecretKey))
	hash.Write(body)
	expectedSignature := "sha256=" + hex.EncodeToString(hash.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (d *Deployer) deploy(payload WebhookPayload) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.deploying {
		log.Printf("⚠️  Deployment already in progress, skipping")
		return
	}

	d.deploying = true
	defer func() { d.deploying = false }()

	log.Printf("🚀 Starting deployment...")
	startTime := time.Now()

	// Change to repository directory
	if err := os.Chdir(d.config.RepoPath); err != nil {
		log.Printf("❌ Failed to change to repo directory: %v", err)
		return
	}

	// Run the deployment script
	cmd := exec.Command("bash", d.config.DeployScript)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DEPLOY_COMMIT=%s", payload.HeadCommit.ID),
		fmt.Sprintf("DEPLOY_MESSAGE=%s", payload.HeadCommit.Message),
		fmt.Sprintf("DEPLOY_AUTHOR=%s", payload.HeadCommit.Author.Name),
		fmt.Sprintf("DEPLOY_BRANCH=%s", d.config.Branch),
	)

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("❌ Deployment failed after %v: %v", duration, err)
		log.Printf("📜 Output: %s", string(output))
		return
	}

	log.Printf("✅ Deployment completed successfully in %v", duration)
	log.Printf("📜 Output: %s", string(output))
}

func (d *Deployer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"deploying": d.deploying,
	})
}

func (d *Deployer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get current git commit
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = d.config.RepoPath
	commitBytes, _ := cmd.Output()
	currentCommit := strings.TrimSpace(string(commitBytes))

	// Get last commit message
	cmd = exec.Command("git", "log", "-1", "--pretty=%s")
	cmd.Dir = d.config.RepoPath
	messageBytes, _ := cmd.Output()
	lastMessage := strings.TrimSpace(string(messageBytes))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ready",
		"repo_path":      d.config.RepoPath,
		"target_branch":  d.config.Branch,
		"current_commit": currentCommit,
		"last_message":   lastMessage,
		"deploying":      d.deploying,
		"timestamp":      time.Now().UTC(),
	})
}
