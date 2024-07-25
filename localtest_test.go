package froach_test

import (
	"context"
	"log"
	"testing"

	"github.com/KarpelesLab/froach"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
)

func TestLocalTest(t *testing.T) {
	// this tests if we actually run a server
	dsn, err := froach.LocalTestServer()
	if err != nil {
		t.Skipf("unable to launch cockroach: %s", err)
		return
	}

	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		panic(err)
	}

	rows, err := conn.Query(context.Background(), "SELECT VERSION()")
	if err != nil {
		panic(err)
	}
	res, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		panic(err)
	}

	log.Printf("version = %v", res)
}
