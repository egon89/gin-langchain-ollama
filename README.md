# Gin LangChain Ollama

## Migration
```bash
# permission if needed
chmod +x migrate.sh

# To run all pending migrations
./migrate.sh up

# To rollback the last migration
./migrate.sh down 1

# To reset the database
./migrate.sh all
```

## sqlc - docker
```bash
docker run --rm -v $(pwd):/src -w /src sqlc/sqlc generate
```
