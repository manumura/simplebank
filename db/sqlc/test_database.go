package db

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestDatabase struct {
	instance testcontainers.Container
}

const (
	postgresImage    = "postgres:14-alpine"
	postgresUser     = "root"
	postgresPassword = "password"
	postgresDatabase = "simplebank"
)

// https://gist.github.com/Eun/221e9cb9190b56578dbb752e5e2a6b2a
// usage:
// testDB := db.NewTestDatabase(t)
// defer testDB.Close(t)
// println(testDB.ConnectionString(t))
func NewTestDatabase() (*TestDatabase, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	req := testcontainers.ContainerRequest{
		Image:        postgresImage,
		ExposedPorts: []string{"5432/tcp"},
		AutoRemove:   true,
		Env: map[string]string{
			"POSTGRES_USER":     postgresUser,
			"POSTGRES_PASSWORD": postgresPassword,
			"POSTGRES_DB":       postgresDatabase,
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		return nil, err
	}

	return &TestDatabase{
		instance: postgres,
	}, nil
}

func (db *TestDatabase) Port() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	p, err := db.instance.MappedPort(ctx, "5432")
	if err != nil {
		return 0, err
	}

	return p.Int(), nil
}

func (db *TestDatabase) ConnectionString() (string, error) {
	p, err := db.Port()
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("postgres://%s:%s@127.0.0.1:%d/%s?sslmode=disable", postgresUser, postgresPassword, p, postgresDatabase)
	return url, nil
}

func (db *TestDatabase) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := db.instance.Terminate(ctx)
	fmt.Println("Error while closing test database: ", err)
}
