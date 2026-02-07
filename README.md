# MongoDB REST API

A high-performance Go-based REST API for MongoDB that provides a unified interface for database operations. Capable of inserting millions of records in seconds through goroutines and retrieving data with full pagination, filtering, sorting, and aggregation support.

[![Go Version](https://img.shields.io/badge/Go-1.17+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![MongoDB](https://img.shields.io/badge/MongoDB-4.4+-47A248?style=flat&logo=mongodb)](https://www.mongodb.com/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- 🚀 **High Performance** - Concurrent inserts and retrievals using goroutines
- 🔍 **Advanced Querying** - Support for complex AND/OR queries with multiple operators
- 📄 **Pagination** - Built-in pagination with configurable page sizes
- 🔄 **Sorting** - Multi-field sorting (ascending/descending)
- 📊 **Aggregation** - GroupBy support with inner pagination
- 🔐 **Authentication** - Optional API key protection
- ⏱️ **Timestamps** - Automatic `created_at` and `updated_at` fields
- 🐳 **Docker Ready** - Full Docker and Docker Compose support

## Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration)
- [API Endpoints](#api-endpoints)
- [Query Syntax](#query-syntax)
- [Examples](#examples)
- [Pre-Built Clients](#pre-built-clients)
- [Testing](#testing)

---

## Quick Start

```bash
# Clone the repository
git clone https://github.com/alexanderthegreat96/mongo-db-api-go.git
cd mongo-db-api-go

# Start with Docker Compose
docker-compose up -d

# API is now available at http://localhost:9874
```

---

## Installation

### Option 1: Docker Compose (Recommended)

```bash
docker-compose up -d
```

This starts both MongoDB and the API server with default configuration.

### Option 2: Manual Installation

```bash
# Install dependencies
go mod tidy

# Run in development mode
go run main.go
```

### Building for Production

The easiest way to build is using the included `build.sh` script, which compiles the application for **all major operating systems and architectures** in one command:

```bash
./build.sh
```

This generates executables in the `/bin` directory:

```
bin/
├── linux/
│   └── mongo-api          # Linux (amd64)
├── mac/
│   └── mongo-api          # macOS (amd64)
└── windows/
    └── mongo-api.exe      # Windows (amd64)
```

### Building Manually (Single Target)

If you only need a specific platform:

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o bin/linux/mongo-api

# macOS
GOOS=darwin GOARCH=amd64 go build -o bin/mac/mongo-api

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/windows/mongo-api.exe
```

---

## Configuration

Create a `.env` file in the project root:

```env
# API Configuration
API_PORT=9777
API_HOST=0.0.0.0
API_KEY=""                    # Optional: Set to enable API key authentication

# MongoDB Connection
MONGO_DB_HOST=localhost
MONGO_DB_PORT=27017
MONGO_DB_NAME=test
MONGO_DB_TABLE=test
MONGO_DB_USERNAME=admin
MONGO_DB_PASSWORD=admin

# Handler Options
HANDLER_USE_TIMESTAMPS=true   # Auto-add created_at/updated_at
HANDLER_DEBUG=true            # Enable debug logging

# Startup Options
WAIT_FOR_MONGO_ON_BOOT=true
WAIT_AT_BOOT=30               # Seconds to wait for MongoDB
```

### API Key Authentication

When `API_KEY` is set, all requests must include the `api_key` header:

```bash
curl -H "api_key: your-secret-key" http://localhost:9777/db/databases
```

---

## API Endpoints

### Database Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/db/databases` | List all databases |
| DELETE | `/db/:db_name/delete` | Delete a database |
| GET | `/db/:db_name/tables` | List all collections in a database |
| DELETE | `/db/:db_name/:table_name/delete` | Delete a collection |

### CRUD Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/db/:db_name/:table_name/select` | Query documents with filtering, sorting, pagination |
| GET | `/db/:db_name/:table_name/get/:mongo_id` | Get a single document by ID |
| GET | `/db/:db_name/:table_name/count` | Count documents (with optional filtering) |
| POST | `/db/:db_name/:table_name/insert` | Insert document(s) |
| PUT | `/db/:db_name/:table_name/update/:mongo_id` | Update a document by ID |
| PUT | `/db/:db_name/:table_name/update-where` | Update documents matching query |
| DELETE | `/db/:db_name/:table_name/delete/:mongo_id` | Delete a document by ID |
| DELETE | `/db/:db_name/:table_name/delete-where` | Delete documents matching query |
| POST | `/db/:db_name/:table_name/custom-query` | Execute raw MongoDB query/pipeline |

---

## Query Syntax

### Query Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `page` | Page number (default: 1) | `page=2` |
| `per_page` | Results per page (default: 10) | `per_page=25` |
| `query_and` | AND conditions | `query_and=status,=,active\|age,>,18` |
| `query_or` | OR conditions | `query_or=role,=,admin\|role,=,moderator` |
| `sort` | Sort order | `sort=[created_at:desc]` |
| `group_by` | Group results by field | `group_by=category` |
| `inner_page` | Page within grouped results | `inner_page=1` |
| `inner_per_page` | Results per group | `inner_per_page=5` |
| `auto_convert_inputs` | Auto-convert types (default: true) | `auto_convert_inputs=false` |

### Supported Operators

| Operator | Description | MongoDB Equivalent | Example |
|----------|-------------|-------------------|---------|
| `=` | Equals | `$eq` | `status,=,active` |
| `!=` | Not equals | `$ne` | `status,!=,deleted` |
| `<>` | Not equals (alt) | `$ne` | `status,<>,deleted` |
| `>` | Greater than | `$gt` | `age,>,18` |
| `<` | Less than | `$lt` | `age,<,65` |
| `>=` | Greater than or equal | `$gte` | `price,>=,100` |
| `<=` | Less than or equal | `$lte` | `price,<=,1000` |
| `like` | Contains (regex) | `$regex` (case-insensitive) | `name,like,john` |
| `not_like` | Not contains | `$not` + `$regex` | `name,not_like,test` |
| `between` | Range (inclusive) | `$gte` + `$lte` | `age,between,[18:65]` |
| `in` | In array | `$in` | `status,in,[active:pending:review]` |
| `not_in` / `nin` | Not in array | `$nin` | `status,nin,[deleted:archived]` |
| `exists` | Field exists | `$exists` | `email,exists,true` |
| `regex` | Regular expression | `$regex` | `email,regex,@gmail.com$` |
| `type` | BSON type check | `$type` | `age,type,int` |
| `mod` | Modulo operation | `$mod` | `quantity,mod,[2:0]` |
| `size` | Array size | `$size` | `tags,size,3` |
| `all` | All elements match | `$all` | `tags,all,[go:mongodb:api]` |

---

## Query Formulation Guide

This section explains how to build queries from scratch. While we provide [PHP](https://github.com/alexanderthegreat96/mongo-api-php-client) and [Python](https://github.com/alexanderthegreat96/mongo-api-python-client) clients, understanding the query format allows you to integrate with any language or use cURL directly.

### Basic Structure

Every query condition follows the format:

```
field,operator,value
```

**Components:**
- **field**: The document field name (supports dot notation for nested fields)
- **operator**: The comparison operator (see table above)
- **value**: The value to compare against

### Combining Multiple Conditions

Use the pipe `|` character to combine multiple conditions:

```
field1,operator1,value1|field2,operator2,value2|field3,operator3,value3
```

**Important:** All conditions in `query_and` are combined with AND logic. All conditions in `query_or` are combined with OR logic.

### Step-by-Step Examples

#### Example 1: Simple Equality

**Goal:** Find users where status equals "active"

```
query_and=status,=,active
```

**Translates to MongoDB:**
```json
{"status": "active"}
```

#### Example 2: Multiple AND Conditions

**Goal:** Find active users over 18 years old in the USA

```
query_and=status,=,active|age,>,18|country,=,USA
```

**Translates to MongoDB:**
```json
{
  "$and": [
    {"status": "active"},
    {"age": {"$gt": 18}},
    {"country": "USA"}
  ]
}
```

#### Example 3: OR Conditions

**Goal:** Find users who are either admin or moderator

```
query_or=role,=,admin|role,=,moderator
```

**Translates to MongoDB:**
```json
{
  "$or": [
    {"role": "admin"},
    {"role": "moderator"}
  ]
}
```

#### Example 4: Combining AND + OR

**Goal:** Find verified users who are either admin or moderator

```
query_and=verified,=,true&query_or=role,=,admin|role,=,moderator
```

**Translates to MongoDB:**
```json
{
  "$and": [
    {"verified": true},
    {
      "$or": [
        {"role": "admin"},
        {"role": "moderator"}
      ]
    }
  ]
}
```

#### Example 5: Nested Fields (Dot Notation)

**Goal:** Find users living in New York City

```
query_and=address.city,=,New York|address.country,=,USA
```

**Translates to MongoDB:**
```json
{
  "$and": [
    {"address.city": "New York"},
    {"address.country": "USA"}
  ]
}
```

#### Example 6: Range Query with BETWEEN

**Goal:** Find products priced between $100 and $500

```
query_and=price,between,[100:500]
```

**Translates to MongoDB:**
```json
{"price": {"$gte": 100, "$lte": 500}}
```

**Syntax:** `[min:max]` - Values separated by colon, wrapped in brackets

#### Example 7: Pattern Matching with LIKE

**Goal:** Find users with Gmail addresses

```
query_and=email,like,@gmail.com
```

**Translates to MongoDB:**
```json
{"email": {"$regex": "@gmail.com", "$options": "i"}}
```

**Note:** `like` is case-insensitive by default

#### Example 8: IN Operator (Multiple Values)

**Goal:** Find orders with status pending, processing, or shipped

```
query_and=status,in,[pending:processing:shipped]
```

**Translates to MongoDB:**
```json
{"status": {"$in": ["pending", "processing", "shipped"]}}
```

**Syntax:** `[value1:value2:value3]` - Values separated by colons

#### Example 9: Checking Field Existence

**Goal:** Find documents where email field exists

```
query_and=email,exists,true
```

**Translates to MongoDB:**
```json
{"email": {"$exists": true}}
```

#### Example 10: Null Values

**Goal:** Find users where deleted_at is null (not deleted)

```
query_and=deleted_at,=,null
```

**Translates to MongoDB:**
```json
{"deleted_at": null}
```

#### Example 11: Complex E-Commerce Query

**Goal:** Find electronics products, in stock, priced $200-$1000, rating > 4, sorted by rating

```
query_and=category,=,electronics|in_stock,=,true|price,>=,200|price,<=,1000|rating,>,4&sort=[rating:desc|price:asc]
```

### Sort Syntax

Wrap in brackets, use `field:direction` format:

```
[field1:direction|field2:direction]
```

**Directions:**
- `asc` - Ascending (A-Z, 0-9, oldest first)
- `desc` - Descending (Z-A, 9-0, newest first)

**Examples:**
```
[created_at:desc]                    # Newest first
[name:asc]                           # Alphabetical
[category:asc|price:desc]            # By category, then expensive first
[rating:desc|reviews:desc|price:asc] # Best rated, most reviews, cheapest
```

### Auto Type Conversion

By default, the API automatically converts string values to appropriate types:

| Input | Converted To |
|-------|-------------|
| `"123"` | `123` (integer) |
| `"45.67"` | `45.67` (float) |
| `"true"` / `"false"` | `true` / `false` (boolean) |
| `"null"` | `null` |
| `"2026-02-07T00:00:00Z"` | Date object |
| `"ObjectId(507f1f77bcf86cd799439011)"` | ObjectId |

To disable auto-conversion (keep everything as strings):
```
?auto_convert_inputs=false
```

### Special Value Formats

| Format | Description | Example |
|--------|-------------|---------|
| `null` | Null value | `deleted_at,=,null` |
| `true` / `false` | Boolean | `active,=,true` |
| `[a:b]` | Range/Array | `age,between,[18:65]` |
| `ObjectId(...)` | MongoDB ObjectId | `user_id,=,ObjectId(507f1f77...)` |

### URL Encoding

When using queries in URLs, remember to URL-encode special characters:

| Character | Encoded |
|-----------|---------|
| `\|` (pipe) | `%7C` |
| `[` | `%5B` |
| `]` | `%5D` |
| `:` | `%3A` |
| space | `%20` or `+` |

**Example (URL-encoded):**
```
?query_and=status%2C%3D%2Cactive%7Cage%2C%3E%2C18
```

Most HTTP clients handle encoding automatically. cURL with quotes usually works without manual encoding.

### Building a Query Checklist

1. ✅ Identify which fields you need to filter on
2. ✅ Choose the appropriate operator for each condition
3. ✅ Decide if conditions should be AND or OR
4. ✅ Format each condition as `field,operator,value`
5. ✅ Join conditions with `|` (pipe)
6. ✅ Use `query_and=` for AND conditions
7. ✅ Use `query_or=` for OR conditions
8. ✅ Add sort with `sort=[field:direction]`
9. ✅ Add pagination with `page=` and `per_page=`

---

## Examples

### List All Databases

```bash
curl http://localhost:9777/db/databases
```

Response:
```json
{
  "status": true,
  "databases": ["myapp", "analytics", "admin"]
}
```

### Select with Filtering and Pagination

```bash
# Find active users over 18, sorted by name, page 2
curl "http://localhost:9777/db/myapp/users/select?query_and=status,=,active|age,>,18&sort=[name:asc]&page=2&per_page=20"
```

Response:
```json
{
  "status": true,
  "code": 200,
  "database": "myapp",
  "table": "users",
  "count": 150,
  "pagination": {
    "total_pages": 8,
    "current_page": 2,
    "next_page": 3,
    "prev_page": 1,
    "last_page": 8,
    "per_page": 20
  },
  "query": {"status": "active", "age": {"$gt": 18}},
  "results": [
    {"_id": "...", "name": "Alice", "age": 25, "status": "active"},
    {"_id": "...", "name": "Bob", "age": 30, "status": "active"}
  ]
}
```

### Complex Multi-Field Query

```bash
# E-commerce product search: electronics, price $500-$2000, in stock, rating > 4
curl "http://localhost:9777/db/shop/products/select?query_and=category,=,electronics|price,>=,500|price,<=,2000|in_stock,=,true|rating,>,4&sort=[rating:desc|price:asc]"
```

### Using AND + OR Conditions Together

```bash
# Find verified users who are either admin or moderator
curl "http://localhost:9777/db/myapp/users/select?query_and=verified,=,true|email_verified,=,true&query_or=role,=,admin|role,=,moderator"
```

### Query with BETWEEN Operator

```bash
# Find users aged 18-65
curl "http://localhost:9777/db/myapp/users/select?query_and=age,between,[18:65]"
```

### Query with LIKE (Pattern Matching)

```bash
# Find users with gmail addresses
curl "http://localhost:9777/db/myapp/users/select?query_and=email,like,@gmail.com"
```

### Query Nested Fields (Dot Notation)

```bash
# Find users in New York
curl "http://localhost:9777/db/myapp/users/select?query_and=address.city,=,NewYork"
```

### Get Single Document by ID

```bash
curl http://localhost:9777/db/myapp/users/get/507f1f77bcf86cd799439011
```

Response:
```json
{
  "status": true,
  "code": 200,
  "database": "myapp",
  "table": "users",
  "result": {
    "_id": "507f1f77bcf86cd799439011",
    "name": "John Doe",
    "email": "john@example.com",
    "age": 30
  }
}
```

### Insert Single Document

```bash
curl -X POST http://localhost:9777/db/myapp/users/insert \
  -d 'payload={"name": "John Doe", "email": "john@example.com", "age": 30}'
```

Response:
```json
{
  "status": true,
  "code": 200,
  "database": "myapp",
  "table": "users",
  "operation": "insert",
  "message": "1 document(s) inserted"
}
```

### Insert Multiple Documents (Batch)

```bash
curl -X POST http://localhost:9777/db/myapp/users/insert \
  -d 'payload=[
    {"name": "Alice", "email": "alice@example.com", "age": 25},
    {"name": "Bob", "email": "bob@example.com", "age": 35},
    {"name": "Charlie", "email": "charlie@example.com", "age": 28}
  ]'
```

### Update by ID

```bash
curl -X PUT http://localhost:9777/db/myapp/users/update/507f1f77bcf86cd799439011 \
  -d 'payload={"name": "John Smith", "age": 31}'
```

### Update by Query (Bulk Update)

```bash
# Set all inactive users to archived
curl -X PUT "http://localhost:9777/db/myapp/users/update-where?query_and=status,=,inactive" \
  -d 'payload={"status": "archived", "archived_at": "2026-02-07T00:00:00Z"}'
```

### Delete by ID

```bash
curl -X DELETE http://localhost:9777/db/myapp/users/delete/507f1f77bcf86cd799439011
```

### Delete by Query (Bulk Delete)

```bash
# Delete all unverified users older than 30 days
curl -X DELETE "http://localhost:9777/db/myapp/users/delete-where?query_and=verified,=,false|created_at,<,2026-01-07"
```

### Count Documents

```bash
# Count active users
curl "http://localhost:9777/db/myapp/users/count?query_and=status,=,active"
```

Response:
```json
{
  "status": true,
  "code": 200,
  "database": "myapp",
  "table": "users",
  "count": 1250
}
```

### Group By with Aggregation

```bash
# Group orders by category with inner pagination
curl "http://localhost:9777/db/analytics/orders/select?group_by=category&inner_page=1&inner_per_page=5&query_and=year,=,2025"
```

Response:
```json
{
  "status": true,
  "code": 200,
  "database": "analytics",
  "table": "orders",
  "count": 5,
  "results": [
    {
      "_id": "electronics",
      "total_records": 150,
      "records": [...],
      "inner_pagination": {
        "total_pages": 30,
        "current_page": 1,
        "per_page": 5
      }
    },
    {
      "_id": "clothing",
      "total_records": 89,
      "records": [...],
      "inner_pagination": {...}
    }
  ]
}
```

### Custom Query (Raw MongoDB Query)

```bash
# Execute raw MongoDB query
curl -X POST "http://localhost:9777/db/myapp/users/custom-query?page=1&per_page=10" \
  -d 'payload={"status": "active", "age": {"$gte": 21}}'
```

### Custom Aggregation Pipeline

```bash
# Execute aggregation pipeline
curl -X POST "http://localhost:9777/db/analytics/orders/custom-query?as_pipeline=true" \
  -d 'payload=[
    {"$match": {"status": "completed"}},
    {"$group": {"_id": "$category", "total": {"$sum": "$amount"}}},
    {"$sort": {"total": -1}}
  ]'
```

---

## Pre-Built Clients

Official client libraries are available for easy integration:

| Language | Repository |
|----------|------------|
| PHP | [mongo-api-php-client](https://github.com/alexanderthegreat96/mongo-api-php-client) |
| Python | [mongo-api-python-client](https://github.com/alexanderthegreat96/mongo-api-python-client) |
| Go | Coming soon |

---

## Testing

The project includes comprehensive unit tests for both the API and driver layers.

```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Run specific package tests
go test ./api -v      # API tests (57 tests)
go test ./driver -v   # Driver tests (86+ tests)
```

### Test Coverage

| Package | Tests | Coverage |
|---------|-------|----------|
| api | 57 | Middleware, endpoints, query parsing |
| driver | 86+ | CRUD operations, helpers, type conversion |

---

## Response Formats

### Success Response

```json
{
  "status": true,
  "code": 200,
  "database": "myapp",
  "table": "users",
  "results": [...]
}
```

### Error Response

```json
{
  "status": false,
  "code": 400,
  "database": "myapp",
  "table": "users",
  "error": "Error description",
  "query": {...}
}
```

### Pagination Response

```json
{
  "pagination": {
    "total_pages": 10,
    "current_page": 1,
    "next_page": 2,
    "prev_page": 0,
    "last_page": 10,
    "per_page": 20
  }
}
```

---

## Error Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Bad Request (invalid input, missing payload) |
| 401 | Unauthorized (invalid/missing API key) |
| 404 | Not Found (document not found) |
| 500 | Internal Server Error (database connection, query error) |

---

## Architecture

```
mongo-db-api-go/
├── main.go              # Application entry point
├── api/
│   └── api.go           # HTTP handlers and routing
├── driver/
│   ├── driver.go        # MongoDB operations (fluent API)
│   ├── helpers.go       # Query parsing, type conversion
│   ├── types.go         # Data structures
│   └── om.go            # Ordered map utilities
├── responses/
│   └── responses.go     # Response structures
├── docker-compose.yml   # Docker configuration
└── Dockerfile           # Container build
```

---

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver)
