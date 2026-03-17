package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

const (
	roleAdmin    = "admin"
	roleReadOnly = "readonly"
)

type webUser struct {
	Username   string `yaml:"username,omitempty"`
	Secret     string `yaml:"secret,omitempty"`
	SecretHash string `yaml:"secret_hash,omitempty"`
	Role       string `yaml:"role,omitempty"`
}

func main() {
	var configPath string
	var username string
	var password string
	var role string
	var cost int

	flag.StringVar(&configPath, "config", "config.yaml", "path to config.yaml")
	flag.StringVar(&username, "username", "", "web UI username")
	flag.StringVar(&password, "password", "", "plaintext password to hash and store")
	flag.StringVar(&role, "role", roleAdmin, "user role: admin or readonly")
	flag.IntVar(&cost, "cost", bcrypt.DefaultCost, "bcrypt cost")
	flag.Parse()

	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	role = normalizeRole(strings.TrimSpace(role))

	if username == "" {
		username = prompt("Username")
	}
	if password == "" {
		password = prompt("Password")
	}

	if username == "" || password == "" {
		log.Fatal("username and password are required")
	}
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		log.Fatalf("cost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
	}

	cfg, mode := loadConfig(configPath)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		log.Fatalf("failed to generate password hash: %v", err)
	}

	updated := false
	users := getUsers(cfg)

	for i := range users {
		if users[i].Username == username {
			users[i].Secret = ""
			users[i].SecretHash = string(hash)
			users[i].Role = role
			updated = true
			break
		}
	}

	if !updated {
		users = append(users, webUser{
			Username:   username,
			Secret:     "",
			SecretHash: string(hash),
			Role:       role,
		})
	}

	setUsers(cfg, users)
	writeConfig(configPath, cfg, mode)
	fmt.Printf("Updated %s: user %q set to role %q\n", configPath, username, role)
}

func prompt(label string) string {
	fmt.Fprintf(os.Stderr, "%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("failed to read %s: %v", strings.ToLower(label), err)
	}
	return strings.TrimSpace(line)
}

func normalizeRole(role string) string {
	if strings.EqualFold(role, roleReadOnly) {
		return roleReadOnly
	}
	return roleAdmin
}

func loadConfig(path string) (map[string]interface{}, os.FileMode) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read %s: %v", path, err)
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("failed to parse %s: %v", path, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		log.Fatalf("failed to stat %s: %v", path, err)
	}

	return cfg, info.Mode().Perm()
}

func writeConfig(path string, cfg map[string]interface{}, mode os.FileMode) {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		log.Fatalf("failed to serialize %s: %v", path, err)
	}

	if err := os.WriteFile(path, data, mode); err != nil {
		log.Fatalf("failed to write %s: %v", path, err)
	}
}

func getUsers(cfg map[string]interface{}) []webUser {
	webSection, ok := cfg["web"].(map[string]interface{})
	if !ok {
		return nil
	}

	rawUsers, ok := webSection["users"].([]interface{})
	if !ok {
		return nil
	}

	users := make([]webUser, 0, len(rawUsers))
	for _, rawUser := range rawUsers {
		userMap, ok := rawUser.(map[string]interface{})
		if !ok {
			continue
		}

		user := webUser{}
		if value, ok := userMap["username"].(string); ok {
			user.Username = value
		}
		if value, ok := userMap["secret"].(string); ok {
			user.Secret = value
		}
		if value, ok := userMap["secret_hash"].(string); ok {
			user.SecretHash = value
		}
		if value, ok := userMap["role"].(string); ok {
			user.Role = value
		}
		users = append(users, user)
	}

	return users
}

func setUsers(cfg map[string]interface{}, users []webUser) {
	webSection, ok := cfg["web"].(map[string]interface{})
	if !ok {
		webSection = make(map[string]interface{})
		cfg["web"] = webSection
	}

	serializedUsers := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		entry := map[string]interface{}{
			"username":    user.Username,
			"secret_hash": user.SecretHash,
			"role":        user.Role,
		}
		if user.Secret != "" {
			entry["secret"] = user.Secret
		}
		serializedUsers = append(serializedUsers, entry)
	}

	webSection["users"] = serializedUsers
}
