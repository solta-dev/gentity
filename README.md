# Gentity - is a codegen simple entity layer implementation 

Implemented methods with database calls via github.com/jackc/pgx/v5

## Features

- [x] Insert one row
- [ ] TODO: Insert multiple rows at one query
- [x] Update one row
- [x] Delete one row
- [ ] TODO: Delete multiple rows at one query
- [x] Fetch rows by simple query
- [x] Fetch all rows of table
- [x] Fetch row by all values of unique index
- [x] Fetch rows by all values of non-unique index

Update and Delete methods use primary key fields as arguments.

Primary key - is key named "primary" or shortest unique key.

All names (of structs and of fields) converted from camelCase to snake_case for database names.  
Table names also use in plural form.

All methods expect context in first parameter. And pgx.Conn object under 'pgconn' name in it.

## How to use

1. In package with your entity structs: `//go:generate go run github.com/dmitry-novozhilov/gentity`
2. With each struct that has table in database:
  a. Before struct: `// gentity`
  b. For each field in unique key specify gentity-tag with it's name: `gentity:"unique=primary_or_something"`
  c. For each field in non-unique key specify gentity-tag with it's name: `gentity:"index=some_index_name"`

## TODO: or not to do =)

* Tests
* Multi-row insert and delete
* Caches
* Fetchers by begin of indexes
* Make create table queries
* Make migration queries
* Automatic analyze table structure from DB
