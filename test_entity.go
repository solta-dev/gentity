package main

import (
	"context"
	"time"
)

//go:generate go run github.com/solta-dev/gentity

type SomeInts int

// type DBExecutorKey string
// type DBExecutor interface {
// 	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
// 	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
// 	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
// }

const (
	Int1 SomeInts = iota
	Int2 SomeInts = iota
	Int3 SomeInts = iota
)

// gentity
type Test struct {
	ID    uint64    `gentity:"unique=primary autoincrement"`
	IntA  int       `gentity:"index=test_int_a_int_b"`
	IntB  SomeInts  `gentity:"index=test_int_a_int_b"`
	StrA  string    `gentity:"unique=test_str_a"`
	TimeA time.Time `gentity:""`
}

func (Test) createTable(ctx context.Context) error {
	pgConn := ctx.Value(DBExecutorKey("dbExecutor")).(DBExecutor)

	if _, err := pgConn.Exec(context.Background(), `CREATE TABLE tests (
		id bigserial PRIMARY KEY,
		int_a integer NOT NULL,
		int_b integer NOT NULL,
		str_a varchar(256) NOT NULL,
		time_a timestamp NOT NULL DEFAULT now()
	)`); err != nil {
		return err
	}

	if _, err := pgConn.Exec(context.Background(), `CREATE INDEX test_int_a_int_b ON tests (int_a, int_b)`); err != nil {
		return err
	}

	if _, err := pgConn.Exec(context.Background(), `CREATE UNIQUE INDEX test_str_a ON tests (str_a)`); err != nil {
		return err
	}

	return nil
}
