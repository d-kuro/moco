package accessor

import (
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type DataBaseAccessor interface {
	Get(addr, user, password string) (*sqlx.DB, error)
	Remove(addr string)
}

// MySQLAccessorConfig contains MySQL connection configurations
type MySQLAccessorConfig struct {
	ConnMaxLifeTime   time.Duration
	ConnectionTimeout time.Duration
	ReadTimeout       time.Duration
}

// MySQLAccessor contains MySQL connection configurations and sqlx.db
type MySQLAccessor struct {
	config *MySQLAccessorConfig
	mu     sync.Mutex
	dbs    map[string]*sqlx.DB
}

// NewMySQLAccessor creates new MySQLAccessor
func NewMySQLAccessor(config *MySQLAccessorConfig) *MySQLAccessor {
	return &MySQLAccessor{
		config: config,
		dbs:    make(map[string]*sqlx.DB),
	}
}

// Get connects a database with specified parameters
func (acc *MySQLAccessor) Get(addr, user, password string) (*sqlx.DB, error) {
	uri := acc.getURI(addr, user, password)

	acc.mu.Lock()
	defer acc.mu.Unlock()

	if _, exists := acc.dbs[uri]; !exists {
		if db, err := acc.connect(uri); err == nil {
			acc.dbs[uri] = db
		} else {
			return nil, err
		}
	}

	db := acc.dbs[uri]
	err := db.Ping()
	if err != nil {
		delete(acc.dbs, uri)
		return nil, err
	}
	return db, nil
}

func (acc *MySQLAccessor) getURI(addr, user, password string) string {
	conf := mysql.NewConfig()
	conf.User = user
	conf.Passwd = password
	conf.Net = "tcp"
	conf.Addr = addr
	conf.Timeout = acc.config.ConnectionTimeout
	conf.ReadTimeout = acc.config.ReadTimeout
	conf.InterpolateParams = true

	return conf.FormatDSN()
}

func (acc *MySQLAccessor) connect(uri string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("mysql", uri)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(acc.config.ConnMaxLifeTime)

	return db, nil
}

// Remove cleans staled connections
func (acc *MySQLAccessor) Remove(addr string) {
	acc.mu.Lock()
	defer acc.mu.Unlock()

	for uri, db := range acc.dbs {
		if !strings.Contains(uri, addr) {
			continue
		}
		db.Close()
		delete(acc.dbs, uri)
	}
}