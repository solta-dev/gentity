![](https://img.shields.io/static/v1?label=Coverage&message=81.4%&color=green)

# Gentity - is a codegen simple entity layer implementation 

Implemented methods with database calls via [github.com/jackc/pgx/v5](https://github.com/jackc/pgx)

## Features

- [x] Insert one row
- [x] Insert multiple rows at one query
- [x] Update one row
- [x] Delete one row
- [x] Delete multiple rows at one query
- [x] Fetch rows by simple query
- [x] Fetch all rows of table
- [x] Fetch row by all values of unique index
- [x] Fetch rows by all values of non-unique index
- [x] Fetch rows via channel

Update and Delete methods use primary key fields as arguments.

Primary key - is key named "primary" or shortest unique key.

All names (of structs and of fields) converted from camelCase to snake_case for database names.  
Table names also use in plural form.

All methods expect context in first parameter. And pgx.Conn object under 'pgconn' name in it.

## How to use

1. In package with your entity structs: `//go:generate go run github.com/solta-dev/gentity`
2. With each struct that has table in database:
   - Before struct: `// gentity`
   - For each field in unique key specify gentity-tag with it's name: `gentity:"unique=primary_or_something"`
   - For each field in non-unique key specify gentity-tag with it's name: `gentity:"index=some_index_name"`
   - For autoincrement field (if it exists) specify gentity-tag `gentity:"autoincrement"`
   - For some functions entity must have a primary key.
   - Example:
```go
// gentity
type Test struct {
	ID    uint64    `gentity:"unique=primary autoincrement"`
	IntA  int       `gentity:"index=test_int_a_int_b"`
	IntB  SomeInts  `gentity:"index=test_int_a_int_b"`
	StrA  string    `gentity:"unique=test_str_a"`
	TimeA time.Time `gentity:""`
}
```

If your tables names in singular form, please specify `--singular` flag in go:generate command

3. Prepare to use entities:
```go
// Get connection from pgx pool
pgConn, err = pgpool.Acquire(ctx)
// Put connection to ctx (you can put pool, connection or transaction)
ctx := context.WithValue(context.Backgrond(), DBExecutorKey("dbExecutor"), pgConn.Conn())
```
4. Insert new row:
```go
e := Test{IntA: 1, IntB: 1, StrA: "a", TimeA: t1}
if err = e.Insert(ctx); err != nil {
    panic(err)
}
fmt.Println("Id of new item is ", e.ID)

```
5. Other use cases see in [test](test_test.go).


## TODO: or not to do =)

* ~~Tests~~
* ~~Multi-row insert and delete~~
* Chunked queries by long arrays
* Change interface to .All() and .One for get resultset instead of channel use (or not because of with channels we can use select{} for read results simultaneous with other jobs)
* On conflict clause in multi row insert
* Custom returns clause in multi row insert
* Fetchers by begin of tree-indexes
* Make create table queries
* Make migration queries
* Caches?
* Automatic analyze table structure from DB


## Alternatives

* [Gorm](https://pkg.go.dev/gorm.io/gorm)
* [Ent](https://pkg.go.dev/entgo.io/ent)
* [Sqlc](https://pkg.go.dev/github.com/sqlc-dev/sqlc)
