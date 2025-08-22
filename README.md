# backend-project
![build-test-lint](https://github.com/DamienAndDustinBackend/backend-project/actions/workflows/built-test-lint.yml/badge.svg?branch=main)

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