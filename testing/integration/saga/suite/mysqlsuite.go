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

	driverSql "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

// MysqlSuite struct for MySQL Suite
type MysqlSuite struct {
	suite.Suite
	*sync.Mutex
	ctx         context.Context
	dbConn      *sql.DB
	dbContainer *mysql.MySQLContainer
}

type dbCredentials struct {
	dbName   string
	user     string
	password string
}

// SetupSuite setup at the beginning of test
func (t *MysqlSuite) SetupSuite() {
	t.Mutex = &sync.Mutex{}
	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	t.ctx = context.Background()
	t.disableLogging()

	dbCreds := &dbCredentials{
		dbName:   "foreman",
		user:     "foreman",
		password: "foreman",
	}

	dbContainer, err := mysql.Run(t.ctx,
		"mysql",
		mysql.WithDatabase(dbCreds.dbName),
		mysql.WithUsername(dbCreds.user),
		mysql.WithPassword(dbCreds.password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("[Server] /usr/sbin/mysqld: ready for connections").
				WithOccurrence(1).
				WithStartupTimeout(30*time.Second)),
	)
	t.Require().NoError(err)
	t.dbContainer = dbContainer

	dbHost, err := dbContainer.Host(t.ctx)
	t.Require().NoError(err)
	dbPort, err := dbContainer.MappedPort(t.ctx, "3306")
	t.Require().NoError(err)

	connectionStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True", dbCreds.user, dbCreds.password, dbHost, dbPort.Int(), dbCreds.dbName)

	if v := os.Getenv("MYSQL_CONNECTION"); v != "" {
		connectionStr = v
	}

	time.Sleep(time.Second * 5)

	t.dbConn, err = sql.Open("mysql", connectionStr)
	require.NoError(t.T(), err)
	err = t.dbConn.Ping()
	require.NoError(t.T(), err)
}

func (t *MysqlSuite) Connection() *sql.DB {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.dbConn
}

// TearDownSuite teardown at the end of test
func (t *MysqlSuite) TearDownSuite() {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	res, err := t.dbConn.Exec("DROP TABLE IF EXISTS saga_history, saga;")
	require.NoError(t.T(), err)
	require.NotNil(t.T(), res)
	require.NoError(t.T(), t.dbConn.Close())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	err = t.dbContainer.Stop(ctx, nil)
	t.Require().NoError(err)
}

func (t *MysqlSuite) disableLogging() {
	nopLogger := NopLogger{}
	require.NoError(t.T(), driverSql.SetLogger(nopLogger))
}

type NopLogger struct {
}

func (l NopLogger) Print(v ...interface{}) {}
