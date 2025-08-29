# backend-project
![build-test-lint](https://github.com/DamienAndDustinBackend/backend-project/actions/workflows/build-test-lint.yaml/badge.svg?branch=main)

## Technology Stack

### Databases
[sqlite](golang-jwt/jwt)
[mysql](https://www.mysql.com/)

### Languages
[go](http://golang.org/)

### Libraries
[gin](https://gin-gonic.com/en/docs/testing/)
[testify](https://github.com/stretchr/testify)
[google/uuid](https://github.com/google/uuid)
[gorm](https://gorm.io/)
[golang-jwt/jwt](https://github.com/golang-jwt/jwt)
[go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)
[mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)
[net/http](https://pkg.go.dev/net/http)

## Endpoints

### Files
GET /files

### Single File
GET /file/id
* returns the file

* DELETE /file/id

POST /file/id

PUT /file/id

### tags
GET /tags
POST /tags
PUT /tags
DELETE /tags

### Files by tag
GET /files/tag/<tag>

file types are tags

### Files by user
GET /files/user/<user_id>

### Return format
{"success": "", "error": "", data: ""}