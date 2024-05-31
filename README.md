![](https://img.shields.io/static/v1?label=Coverage&message=82.2%&color=green)

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
  a. Before struct: `// gentity`
  b. For each field in unique key specify gentity-tag with it's name: `gentity:"unique=primary_or_something"`
  c. For each field in non-unique key specify gentity-tag with it's name: `gentity:"index=some_index_name"`
  d. For autoincrement field (if it exists) specify gentity-tag `gentity:"autoincrement"`
  e. For some functions entity must have a primary key.

## TODO: or not to do =)

* ~~Tests~~
* ~~Multi-row insert and delete~~
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
