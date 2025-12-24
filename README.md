# billing3



## Development Setup

1. Install golang
2. Copy and modify `.env.example` to `.env`
3. Start postgresql server
4. Start redis server
5. `go run .`

```
docker run -d --name billing3-pg -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=postgres -p 5432:5432 postgres:18
docker run -d --name billing3-redis -p 6379:6379 redis:8
```


```
sqlc generate
```

```

```