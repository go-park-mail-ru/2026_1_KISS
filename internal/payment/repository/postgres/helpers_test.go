package postgres

import (
	"database/sql"
	"time"
)

func anyTime() time.Time { return time.Now() }

func sqlNoRows() error { return sql.ErrNoRows }
