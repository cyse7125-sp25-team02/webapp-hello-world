# Health Check API

A simple health check endpoint implementation in Go that records each check in a MySQL database.

## Table of Contents

- [Health Check API](#health-check-api)
  - [Table of Contents](#table-of-contents)
  - [Requirements](#requirements)
  - [Project Structure](#project-structure)
  - [Setup](#setup)
    - [1. Database Setup](#1-database-setup)
    - [2. Environment Configuration](#2-environment-configuration)
    - [3. Install Dependencies](#3-install-dependencies)
    - [4. Run the Application](#4-run-the-application)
  - [API Documentation](#api-documentation)
    - [Health Check Endpoint](#health-check-endpoint)
      - [Features](#features)
      - [Response Status Codes](#response-status-codes)
      - [Response Headers](#response-headers)
  - [Development Guide](#development-guide)
    - [Code Structure](#code-structure)
    - [Database Schema](#database-schema)
  - [Testing](#testing)
    - [Manual Testing with curl](#manual-testing-with-curl)
    - [Expected Behaviors](#expected-behaviors)
  - [License](#license)

## Requirements

- Go 1.21 or higher
- MySQL 8.0 or higher

## Project Structure

```
webapp-hello-world/
├── cmd/
│ └── server/
│ └── main.go # Application entry point
├── internal/
│ ├── config/
│ │ └── config.go # Configuration management
│ ├── handler/
│ │ └── health.go # HTTP request handler
│ ├── model/
│ │ └── health.go # Database models
│ └── database/
│ └── mysql.go # Database connection
├── migrations/
│ └── 001_create_health_check_table.sql
├── .env # Environment variables
├── go.mod # Go module definition
└── go.sum # Module checksums
```

## Setup

### 1. Database Setup

```
CREATE DATABASE healthdb;
USE healthdb;

CREATE TABLE health_check (
check_id BIGINT AUTO_INCREMENT PRIMARY KEY,
datetime DATETIME NOT NULL
);
```

### 2. Environment Configuration

Create a `.env` file in the project root:

```
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=healthdb
```

### 3. Install Dependencies

- Initialize Go module

```
go mod init webapp-hello-world
```

- Install required packages

```
  go get github.com/go-sql-driver/mysql
  go get github.com/joho/godotenv
```

### 4. Run the Application

```
go run cmd/server/main.go
```

## API Documentation

### Health Check Endpoint

GET /healthz

#### Features

- Only accepts GET requests
- No request parameters or payload allowed
- Records check timestamp in UTC
- Returns appropriate HTTP status codes
- Includes cache control headers

#### Response Status Codes

| Status Code | Description                                          |
| ----------- | ---------------------------------------------------- |
| 200         | OK - Health check successful                         |
| 400         | Bad Request - Request contains payload or parameters |
| 405         | Method Not Allowed - Non-GET requests                |
| 503         | Service Unavailable - Database connection failed     |

#### Response Headers

Cache-Control: no-cache, no-store, must-revalidate
Pragma: no-cache
Expires: 0
Content-Type: application/json

## Development Guide

### Code Structure

1. **main.go**: Application entry point

   - Server initialization
   - Route configuration
   - Database connection setup

2. **config.go**: Configuration management

   - Environment variable loading
   - Default configuration values
   - Configuration structure

3. **health.go (handler)**: Request handling

   - Request validation
   - Response formatting
   - Error handling

4. **mysql.go**: Database operations
   - Connection management
   - Query execution
   - Error handling

### Database Schema

```
CREATE TABLE health_check (
check_id BIGINT AUTO_INCREMENT PRIMARY KEY,
datetime DATETIME NOT NULL
);
```

## Testing

### Manual Testing with curl

- Successful health check
  `curl -v http://localhost:8080/healthz`

- Test invalid method
  `curl -v -X POST http://localhost:8080/healthz`

- Test with query parameters (should fail)
  `curl -v "http://localhost:8080/healthz?param=value"`

- Test with payload (should fail)
  `curl -v -X GET -d '{"test":"data"}' http://localhost:8080/healthz`

### Expected Behaviors

1. Valid GET request:

   - Returns 200 OK
   - Empty response body
   - Record inserted in database

2. Invalid requests:
   - POST/PUT/DELETE: 405 Method Not Allowed
   - GET with payload: 400 Bad Request
   - GET with parameters: 400 Bad Request

## License

This project is licensed under the MIT License - see the LICENSE file for details
