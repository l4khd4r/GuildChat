// Command migrate applies, rolls back, or inspects the database schema.
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate down [steps]   # steps omitted rolls everything back
//	go run ./cmd/migrate version
//	go run ./cmd/migrate force <version>
//
// `up` also runs automatically when the server boots; this command exists for
// rollbacks, inspection, and recovering a dirty schema.
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/l4khd4r/GuildChat/internal/config"
	"github.com/l4khd4r/GuildChat/internal/database"
)

func main() {
	log.SetFlags(0)

	args := os.Args[1:]
	if len(args) == 0 {
		usage()
	}

	cfg := config.Load()
	dbCfg := database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Name:     cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	}

	switch args[0] {
	case "up":
		if err := database.MigrateUp(dbCfg); err != nil {
			log.Fatal(err)
		}
		report(dbCfg)

	case "down":
		steps := 0
		if len(args) > 1 {
			n, err := strconv.Atoi(args[1])
			if err != nil || n < 0 {
				log.Fatalf("down: steps must be a non-negative integer, got %q", args[1])
			}
			steps = n
		}
		if err := database.MigrateDown(dbCfg, steps); err != nil {
			log.Fatal(err)
		}
		report(dbCfg)

	case "version":
		report(dbCfg)

	case "force":
		if len(args) < 2 {
			log.Fatal("force: needs a version, e.g. `force 1`")
		}
		v, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatalf("force: %q is not a version number", args[1])
		}
		if err := database.MigrateForce(dbCfg, v); err != nil {
			log.Fatal(err)
		}
		report(dbCfg)

	default:
		usage()
	}
}

func report(dbCfg database.Config) {
	version, dirty, err := database.MigrateVersion(dbCfg)
	if err != nil {
		log.Fatal(err)
	}
	if version == 0 {
		fmt.Println("schema version: none (no migrations applied)")
		return
	}
	state := "clean"
	if dirty {
		state = "DIRTY — a migration failed part way; inspect the schema, then `force` a version"
	}
	fmt.Printf("schema version: %d (%s)\n", version, state)
}

func usage() {
	log.Fatal("usage: migrate up | down [steps] | version | force <version>")
}
