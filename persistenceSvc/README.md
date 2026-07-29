persistenceSvc
===============

Environment
-----------
- `CORS_ALLOW_ORIGINS` (optional): semicolon-separated origins allowed by CORS. Example:

  `CORS_ALLOW_ORIGINS=http://example.com;http://localhost:5173`

  Use `*` to allow all origins (not recommended for production).

- `MONGO_URI` (optional): MongoDB connection URI. Replace with your cluster/user.

- `MATCHING_SVC_URL` (optional): matching service base URL. Defaults to
  `http://localhost:9020`.

Example
-------
Copy `.env.example` to `.env` and edit values:

```sh
cp .env.example .env
# edit .env as needed
export $(grep -v '^#' .env | xargs)
go run main.go
```

Notes
-----
- `main.go` has been updated to read `CORS_ALLOW_ORIGINS` and fall back to a safe default when the env var is not set.

Authorization
-------------
Authentication and authorization are separate:

- `AuthMiddleware` verifies the JWT, loads the current role from
  `project.usersdb`, and stores the user's identity and role.
- Route middleware checks whether the role grants the requested action.
- The application service enforces `own` scope against a profile's `ownerId`
  or a job's `creatorId`.

New records receive their ownership field from the JWT subject; client-provided
owner IDs are ignored. Existing records should be backfilled with `ownerId` or
`creatorId` before users with `own` scope access them.
