package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/Garv25567/Skribbl/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// to do
// [x] Create a server
// [x]connection upgrade
// [x]  create a client and help him get into a room
// [x] make a room of 6 people or more then get that in a server
// [x] go channel run to accept loop

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)

	createNewWSServer(dbQueries)
}
