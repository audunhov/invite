package app

import (
	"database/sql"
	"invite/db"
	"invite/email"
	"invite/models"
)

type App struct {
	Queries      *db.Queries
	DB           *sql.DB
	EmailService *email.Service
}

var _ models.Inviter = (*App)(nil)

func New(dbConn *sql.DB, queries *db.Queries, emailService *email.Service) *App {
	return &App{
		Queries:      queries,
		DB:           dbConn,
		EmailService: emailService,
	}
}
