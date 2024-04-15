package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/techschool/simplebank/util"
)

// https://medium.com/@kelseyhightower/12-fractured-apps-1080c73d481c
// docker run --name postgres -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -d postgres:14-alpine
// docker run --name redis -p 6379:6379 -d redis:alpine
func OpenDBConnection(config util.Config) (*pgxpool.Pool, error) {
	connPool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Info().Err(err).Msg("cannot open db connection")
		return nil, err
	}

	var dbError error
	maxAttempts := 10
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		dbError = connPool.Ping(context.Background())
		if dbError == nil {
			break
		}
		log.Info().Msgf("cannot connect to db (%d): %s\n", attempt, dbError)
		// sleep with exponential backoff
		time.Sleep(time.Duration(attempt*attempt) * time.Second)
	}

	return connPool, dbError
}
