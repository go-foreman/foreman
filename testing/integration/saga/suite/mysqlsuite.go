package suite

import (
	"database/sql"
	"fmt"
	driverSql "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"os"
)

// MysqlSuite struct for MySQL Suite
type MysqlSuite struct {
	suite.Suite
	dbConn        *sql.DB
}

// SetupSuite setup at the beginning of test
func (s *MysqlSuite) SetupSuite() {
	disableLogging()

	connectionStr := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True", "foreman", "foreman", "127.0.0.1:3306", "foreman")

	if v := os.Getenv("MYSQL_CONNECTION"); v != "" {
		connectionStr = v
	}

	var err error
	s.dbConn, err = sql.Open("mysql", connectionStr)
	require.NoError(s.T(), err)
	err = s.dbConn.Ping()
	require.NoError(s.T(), err)
}

func (s *MysqlSuite) Connection() *sql.DB {
	return s.dbConn
}

// TearDownSuite teardown at the end of test
func (s *MysqlSuite) TearDownSuite() {
	res, err := s.dbConn.Exec("DROP TABLE saga_history, saga;")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), res)
	require.NoError(s.T(), s.dbConn.Close())
}

func disableLogging() {
	nopLogger := NopLogger{}
	driverSql.SetLogger(nopLogger)
}

type NopLogger struct {
}

func (l NopLogger) Print(v ...interface{}) {}