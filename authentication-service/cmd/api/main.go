package main

import (
	"database/sql"
	"log"

	"github.com/HendricksK/authentication/data"
)

const webPort = "80"

type Config struct {
	DB     *sql.DB
	Models data.Models
}

func main() {
	log.Println(("Starting authentication service"))
}
