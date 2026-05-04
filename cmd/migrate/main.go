//go:build ignore

package main

import (
	"context"
	"log"
	"os"

	"ariga.io/atlas/sql/migrate"
	atlasmigrate "entgo.io/ent/dialect/sql/schema"

	entmigrate "github.com/aescanero/dago/ent/migrate"

	_ "github.com/lib/pq"
)

func main() {
	dir, err := migrate.NewLocalDir("./migrations")
	if err != nil {
		log.Fatalf("new local dir: %v", err)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://dago:dago@localhost:5432/dago?sslmode=disable"
	}

	name := "init_graph_node_execution"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	ctx := context.Background()
	opts := []atlasmigrate.MigrateOption{
		atlasmigrate.WithDir(dir),
	}
	if err := atlasmigrate.Diff(
		ctx, dsn, name, entmigrate.Tables, opts...,
	); err != nil {
		log.Fatalf("named diff: %v", err)
	}
	log.Println("Migration generated:", name)
}
