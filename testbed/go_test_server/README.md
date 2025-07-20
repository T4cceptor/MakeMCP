# Go Test Server for MakeMCP

A Go server application using the Fuego framework with automatic OpenAPI v3.1 spec generation.
It's intended as a test and sample server for MakeMCP to generate valid configs 

## Requirements

- Go 1.24. (tested)

## Installation

Install dependencies:

```bash
make install
```

## Running the Server

Start the development server:

```bash
make run
```

The server will start on `http://localhost:8120` by default.
You can modify the `port` variable in `main.go` to change the port.

## OpenAPI Documentation

The server automatically generates OpenAPI documentation available at:
- Swagger UI: `http://localhost:8120/swagger/index.html`
- OpenAPI JSON: `http://localhost:8120/swagger/openapi.json`

## Available Commands

- `make install` - Install project dependencies
- `make run` - Start the development server
- `make build` - Build the server binary
- `make clean` - Clean build artifacts and database

## API Endpoints

- `GET /` - Welcome message
- `GET /users` - List all users
- `POST /users` - Create a new user
- `GET /users/{id}` - Get user by ID
- `GET /users/by_email/?email=...` - Get user by email
- `PATCH /users/{id}` - Update user
- `DELETE /users/{id}` - Delete user