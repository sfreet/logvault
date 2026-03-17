package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	var password string
	var cost int

	flag.StringVar(&password, "password", "", "plaintext password to hash")
	flag.IntVar(&cost, "cost", bcrypt.DefaultCost, "bcrypt cost")
	flag.Parse()

	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		log.Fatalf("cost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
	}

	if password == "" {
		fmt.Fprint(os.Stderr, "Password: ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("failed to read password: %v", err)
		}
		password = strings.TrimSpace(line)
	}

	if password == "" {
		log.Fatal("password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		log.Fatalf("failed to generate password hash: %v", err)
	}

	fmt.Println(string(hash))
}
