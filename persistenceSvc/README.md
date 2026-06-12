persistenceSvc
===============

Environment
-----------
- `CORS_ALLOW_ORIGINS` (optional): semicolon-separated origins allowed by CORS. Example:

  `CORS_ALLOW_ORIGINS=http://example.com;http://localhost:5173`

  Use `*` to allow all origins (not recommended for production).

- `MONGO_URI` (optional): MongoDB connection URI. Replace with your cluster/user.

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
