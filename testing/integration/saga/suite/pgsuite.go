package suite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	_ "github.com/jackc/pgx/v4/stdlib"
)

type PgSuite struct {
	suite.Suite
	dbConn *sql.DB
}

// SetupSuite setup at the beginning of test
func (s *PgSuite) SetupSuite() {
	// urlExample := "postgres://username:password@localhost:5432/database_name"
	connectionStr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", "foreman", "foreman", "127.0.0.1:5432", "foreman")

	if v := os.Getenv("PG_CONNECTION"); v != "" {
		connectionStr = v
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var err error
	s.dbConn, err = sql.Open("pgx", connectionStr)
	require.NoError(s.T(), err)
	err = s.dbConn.PingContext(ctx)
	require.NoError(s.T(), err)
}

func (s *PgSuite) Connection() *sql.DB {
	return s.dbConn
}

// TearDownSuite teardown at the end of test
func (s *PgSuite) TearDownSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	res, err := s.dbConn.ExecContext(ctx, "DROP TABLE IF EXISTS saga_history, saga;")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), res)
	require.NoError(s.T(), s.dbConn.Close())
}
