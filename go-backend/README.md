# Go Backend

This service is the data source for the React + Node.js test application. It exposes JSON REST endpoints for users, tasks, stats, and health checks.

## Setup

1. Make sure Go 1.21 or higher is installed.
2. Start the service:

```bash
go run .
```

The service runs on `http://localhost:8080` by default. Set `PORT` to override the port.

## API Endpoints

### Health

- `GET /health`

### Users

- `GET /api/users`
- `GET /api/users/{id}`
- `POST /api/users`

Create user body:

```json
{
  "name": "Ada Lovelace",
  "email": "ada@example.com",
  "role": "developer"
}
```

`name`, `email`, and `role` are required. Email uses basic format validation. Successful creates return `201` and the created user.

### Tasks

- `GET /api/tasks`
- `GET /api/tasks?status=pending&userId=1`
- `POST /api/tasks`
- `PUT /api/tasks/{id}`

Create task body:

```json
{
  "title": "Ship API",
  "status": "pending",
  "userId": 1
}
```

Update task body supports partial updates:

```json
{
  "status": "completed"
}
```

Valid task statuses are `pending`, `in-progress`, and `completed`. `userId` must reference an existing user. Missing tasks return `404`; validation errors return `400`.

### Stats

- `GET /api/stats`

## Logging

Every request is logged with method, path, response status, and duration:

```text
request method=POST path=/api/tasks status=201 duration=1.2ms
```

Request and server errors are logged with method/path context.

## Testing

Run the Go tests from this directory:

```powershell
$env:GOCACHE='D:\go_test\.gocache'; go test ./... -cover
```

Current coverage is above 70% and includes datastore unit tests plus HTTP endpoint tests.
