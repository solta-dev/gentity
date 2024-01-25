package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type tracer struct {
	t *testing.T
}

func (tracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	log.Printf("SQL: %s, args: %+v\n", data.SQL, data.Args)
	return context.WithValue(ctx, "pg_query_start_ts", time.Now())
}
func (t tracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	log.Printf("command_tag: %s, err: %+v, rows_affected: %d, duration: %s\n", data.CommandTag.String(), data.Err, data.CommandTag.RowsAffected(), time.Since(ctx.Value("pg_query_start_ts").(time.Time)))
	if data.Err != nil {
		t.t.Fatal(data.Err)
	}
}

func getFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

func TestMain(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pgPort, err := getFreePort()
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.CommandContext(ctx,
		"docker", "run",
		"--env", "POSTGRES_PASSWORD=gentity",
		"--env", "POSTGRES_USER=gentity",
		"--env", "POSTGRES_DB=gentity",
		"--memory", "100M", "--publish", fmt.Sprintf("%d:5432", pgPort),
		"--rm",
		"postgres:15",
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	fmt.Println("started docker", cmd.Process.Pid)
	defer func() {
		if err := syscall.Kill(cmd.Process.Pid, syscall.SIGINT); err != nil {
			t.Fatal(err)
		}
		fmt.Println("killed docker", cmd.Process.Pid)
		if err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}
		fmt.Println("waited docker", cmd.Process.Pid)
	}()

	pgconf, err := pgxpool.ParseConfig(fmt.Sprintf("host=127.0.0.1 user=gentity password=gentity dbname=gentity port=%d sslmode=disable", pgPort))
	if err != nil {
		t.Fatal(err)
	}
	pgconf.ConnConfig.Tracer = tracer{t: t}
	pgpool, err := pgxpool.NewWithConfig(context.Background(), pgconf)
	if err != nil {
		t.Fatal(err)
	}

	pgAwaitingStart := time.Now()
	var pgConn *pgxpool.Conn
	for {
		pgConn, err = pgpool.Acquire(ctx)

		if err != nil {
			if time.Since(pgAwaitingStart) > 10*time.Second {
				t.Fatal(err)
			} else {
				time.Sleep(100 * time.Millisecond)
				continue
			}
		} else {
			break
		}
	}

	ctx = context.WithValue(ctx, "pgconn", *pgConn.Conn())

	err = Test{}.createTable(ctx)
	if err != nil {
		t.Fatal(err)
	}

	os.Setenv("GOFILE", "test_entity.go")
	main()

	es, err := Test{}.GetAll(ctx)
	if err != nil {
		t.Error(err)
	}
	if len(es) != 0 {
		t.Error("len(es) != 0, but no inserts were made")
	}

	t1 := time.Now().Truncate(time.Microsecond).UTC()

	// Simple insert
	e1 := Test{IntA: 1, IntB: 1, StrA: "a", TimeA: t1}
	if err = e1.Insert(ctx); err != nil {
		t.Error(err)
	}
	// Auto get id
	if diff := deep.Equal(e1, Test{ID: 1, IntA: 1, IntB: 1, StrA: "a", TimeA: t1}); diff != nil {
		t.Error(diff)
	}

	e2 := Test{IntA: 2, IntB: 2, StrA: "b", TimeA: t1}
	if err = e2.Insert(ctx); err != nil {
		t.Error(err)
	}
	if diff := deep.Equal(e2, Test{ID: 2, IntA: 2, IntB: 2, StrA: "b", TimeA: t1}); diff != nil {
		t.Error(diff)
	}

	e3 := Test{ID: 33, IntA: 2, IntB: 2, StrA: "c", TimeA: t1}
	if err = e3.Insert(ctx); err != nil {
		t.Error(err)
	}
	if diff := deep.Equal(e3, Test{ID: 33, IntA: 2, IntB: 2, StrA: "c", TimeA: t1}); diff != nil {
		t.Error(diff)
	}

	// Get all
	es, err = Test{}.GetAll(ctx)
	if err != nil {
		t.Error(err)
	}
	if diff := deep.Equal(es, []Test{e1, e2, e3}); diff != nil {
		t.Error(diff)
	}

	// Return and update vals
	e4 := Test{IntA: 4, IntB: 4, StrA: "d"}
	if err = e4.Insert(ctx, InsertOption{ReturnAndUpdateVals: true}); err != nil {
		t.Error(err)
	}
	var e4fromDB *Test
	e4fromDB, err = Test{}.GetByPrimary(ctx, e4.ID) // Test get by primary index
	if err != nil {
		t.Error(err)
	}
	if !e4.TimeA.Equal(e4fromDB.TimeA) {
		t.Error("Setted in DB time wasn't returned")
	}

	// On conflict
	e5 := Test{IntA: 5, IntB: 5, StrA: "a", TimeA: t1}
	if err = e5.Insert(ctx, InsertOption{OnConflictStatement: "(str_a) DO UPDATE SET int_a = tests.int_a + EXCLUDED.int_a"}); err != nil {
		t.Error(err)
	}
	var e1fromDB *Test
	e1fromDB, err = Test{}.GetByPrimary(ctx, e1.ID)
	if err != nil {
		t.Error(err)
	}
	if diff := deep.Equal(*e1fromDB, Test{ID: e1.ID, IntA: 6, IntB: 1, StrA: "a", TimeA: t1}); diff != nil {
		t.Error(diff)
	}

	// Get by non unique index
	e23, err := Test{}.GetByTestIntAIntB(ctx, 2, 2)
	if err != nil {
		t.Error(err)
	}
	if diff := deep.Equal(e23, []Test{e2, e3}); diff != nil {
		t.Error(diff)
	}

	// Get by unique index
	var e1fromDB2 *Test
	e1fromDB2, err = Test{}.GetByTestStrA(ctx, "a")
	if err != nil {
		t.Error(err)
	}
	if diff := deep.Equal(e1fromDB2, e1fromDB); diff != nil {
		t.Error(diff)
	}

	// Multi get by unique index
	e12, err := Test{}.MultiGetByPrimary(ctx, []uint64{1, 2})
	if err != nil {
		t.Error(err)
	}
	if e12[0].ID > e12[1].ID {
		e12[0], e12[1] = e12[1], e12[0]
	}
	if diff := deep.Equal(e12, []Test{*e1fromDB, e2}); diff != nil {
		t.Error(diff)
	}
	e12, err = Test{}.MultiGetByTestStrA(ctx, []string{"a", "b"})
	if err != nil {
		t.Error(err)
	}
	if e12[0].ID > e12[1].ID {
		e12[0], e12[1] = e12[1], e12[0]
	}
	if diff := deep.Equal(e12, []Test{*e1fromDB, e2}); diff != nil {
		t.Error(diff)
	}

	// Update
	e1.IntB = 111
	if err = e1.Update(ctx); err != nil {
		t.Error(err)
	}
	e1fromDB, err = Test{}.GetByPrimary(ctx, 1)
	if err != nil {
		t.Error(err)
	}
	// IntA == 1 because of gentity updates all fields, not changed only.
	if diff := deep.Equal(*e1fromDB, Test{ID: 1, IntA: 1, IntB: 111, StrA: "a", TimeA: t1}); diff != nil {
		t.Error(diff)
	}

	// Delete
	if err = e1.Delete(ctx); err != nil {
		t.Error(err)
	}
	e1fromDB, err = Test{}.GetByPrimary(ctx, 1)
	if err != nil {
		t.Error(err)
	}
	if e1fromDB != nil {
		t.Error("e1 was not deleted")
	}

	// Multi insert
	var esRefs = Tests([]*Test{
		{IntA: 6, IntB: 6, StrA: "e", TimeA: t1},
		{IntA: 6, IntB: 6, StrA: "f", TimeA: t1},
		{IntA: 6, IntB: 6, StrA: "g", TimeA: t1},
	})
	if err = esRefs.Insert(ctx); err != nil {
		t.Error(err)
	}
	es, err = Test{}.GetByTestIntAIntB(ctx, 6, 6)
	if err != nil {
		t.Error(err)
	}
	for i := range es {
		es[i].ID = 0 // Because of we don't get autoincrement values in multi insert
	}
	if diff := deep.Equal(es, []Test{*esRefs[0], *esRefs[1], *esRefs[2]}); diff != nil {
		t.Error(diff)
	}

	esRefs = Tests([]*Test{
		{ID: 8, IntA: 9, IntB: 9, StrA: "h", TimeA: t1},
		{ID: 9, IntA: 9, IntB: 9, StrA: "i", TimeA: t1},
		{ID: 10, IntA: 9, IntB: 9, StrA: "j", TimeA: t1},
	})
	if err = esRefs.Insert(ctx); err != nil {
		t.Error(err)
	}
	es, err = Test{}.GetByTestIntAIntB(ctx, 9, 9)
	if err != nil {
		t.Error(err)
	}
	if diff := deep.Equal(es, []Test{*esRefs[0], *esRefs[1], *esRefs[2]}); diff != nil {
		t.Error(diff)
	}

	// Multi delete
	esRefs = Tests([]*Test{esRefs[0], esRefs[2]})
	if err = esRefs.Delete(ctx); err != nil {
		t.Error(err)
	}
	es, err = Test{}.GetByTestIntAIntB(ctx, 9, 9)
	if err != nil {
		t.Error(err)
	}
	if diff := deep.Equal(es, []Test{{ID: 9, IntA: 9, IntB: 9, StrA: "i", TimeA: t1}}); diff != nil {
		t.Error(diff)
	}

}
