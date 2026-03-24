# Gin LangChain Ollama

## Ollama
To run Ollama with Docker, you can use this repository: [egon89/docker-mono-repo/ollama](https://github.com/egon89/docker-mono-repo/tree/main/ollama)

## To run the application
Start **Tika** and **Postgres** using Docker Compose:
```bash
docker-compose up -d
```

Run the migrations to set up the database schema:
```bash
./migrate.sh up
```

Start the Go application:
```bash
go run cmd/main.go
```

The RAG tables will be created in the Postgres database and populated with the extracted content from the files in the `RAG_PATH` directory.

The application will be accessible at [http://localhost:8080/public/chat](http://localhost:8080/public/chat).


## Migration
We are using [golang-migrate/migrate](https://github.com/golang-migrate/migrate) for database migrations. The `migrate.sh` script is a simple wrapper to run the migration commands using Docker.
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
To generate the Go code from the SQL queries using sqlc, you can run the following command in the terminal:

```bash
docker run --rm -v $(pwd):/src -w /src sqlc/sqlc generate
```
