package suite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PgSuite struct {
	suite.Suite
	*sync.Mutex
	ctx         context.Context
	dbConn      *sql.DB
	dbContainer *postgres.PostgresContainer
}

// SetupSuite setup at the beginning of test
func (t *PgSuite) SetupSuite() {
	// urlExample := "postgres://username:password@localhost:5432/database_name"
	t.Mutex = &sync.Mutex{}
	t.ctx = context.Background()
	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	dbCreds := &dbCredentials{
		dbName:   "foreman",
		user:     "foreman",
		password: "foreman",
	}

	dbContainer, err := postgres.Run(t.ctx,
		"docker.io/postgres:16-alpine",
		postgres.WithDatabase(dbCreds.dbName),
		postgres.WithUsername(dbCreds.user),
		postgres.WithPassword(dbCreds.password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t.T(), err)
	t.dbContainer = dbContainer
	dbHost, err := dbContainer.Host(t.ctx)
	t.Require().NoError(err)
	dbPort, err := dbContainer.MappedPort(t.ctx, "5432")
	t.Require().NoError(err)

	connectionStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbCreds.user, dbCreds.password, dbHost, dbPort.Int(), dbCreds.dbName)

	if v := os.Getenv("PG_CONNECTION"); v != "" {
		connectionStr = v
	}

	time.Sleep(time.Second * 5)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	t.dbConn, err = sql.Open("pgx", connectionStr)
	require.NoError(t.T(), err)
	err = t.dbConn.PingContext(ctx)
	require.NoError(t.T(), err)
}

func (t *PgSuite) Connection() *sql.DB {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.dbConn
}

// TearDownSuite teardown at the end of test
func (t *PgSuite) TearDownSuite() {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	res, err := t.dbConn.ExecContext(ctx, "DROP TABLE IF EXISTS saga_history, saga;")
	require.NoError(t.T(), err)
	require.NotNil(t.T(), res)
	require.NoError(t.T(), t.dbConn.Close())

	err = t.dbContainer.Stop(ctx, nil)
	require.NoError(t.T(), err)
}
