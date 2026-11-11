//go:build acceptance

package harness

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	postgresImage = "postgres:14.5-alpine"
	postgresUser  = "silvioubaldino"

	EnvDatabaseURL = "ACCEPTANCE_DATABASE_URL"
)

func startPostgres(ctx context.Context) (string, func(), error) {
	if dsn := os.Getenv(EnvDatabaseURL); dsn != "" {
		return dsn, func() {}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	container, err := postgres.Run(ctx, postgresImage,
		postgres.WithDatabase("acceptance"),
		postgres.WithUsername(postgresUser),
		postgres.WithPassword("acceptance"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return "", nil, fmt.Errorf("cannot start the Postgres container (is Docker running? "+
			"set %s to use an existing database): %w", EnvDatabaseURL, err)
	}

	cleanup := func() {
		_ = testcontainers.TerminateContainer(container)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("cannot read the Postgres connection string: %w", err)
	}

	return dsn, cleanup, nil
}
