package session

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySqlSessionRepository struct {
	DSN string
	DB  *sql.DB
}

func (r *MySqlSessionRepository) SetupRepo() error {
	db, err := sql.Open("mysql", r.DSN)
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(10)

	err = db.Ping()
	if err != nil {
		return err
	}
	r.DB = db

	return nil
}

func NewMySqlSessionRepository(dsn string) (SessionRepository, error) {
	repo := &MySqlSessionRepository{DSN: dsn}
	err := repo.SetupRepo()
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *MySqlSessionRepository) SaveSession(session *Session, duration time.Duration) error {
	var exists int8
	ctx := context.TODO()

	tx, txerr := r.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: false})
	if txerr != nil {
		return txerr
	}
	defer tx.Rollback()

	row := tx.QueryRow("SELECT EXISTS(SELECT * FROM sessions WHERE id = ?) AS 'exists'", session.ID)
	err := row.Scan(&exists)
	if err != nil {
		return err
	}
	if exists == 1 {
		return ErrorSessionAlreadyExists
	}

	_, inserterr := tx.Exec(
		`INSERT INTO sessions (id, user_id) VALUES (?, UUID_TO_BIN(?))`,
		session.ID,
		session.UserID,
	)
	if inserterr != nil {
		return inserterr
	}

	commiterr := tx.Commit()
	if commiterr != nil {
		return commiterr
	}

	return nil
}

func (r *MySqlSessionRepository) ValidateSession(session *Session) (bool, error) {
	fmt.Printf("sessionid: %v\n", session.ID)
	sess := &Session{}
	row := r.DB.QueryRow("SELECT id AS ID, BIN_TO_UUID(user_id) AS UserID FROM sessions WHERE id = ?", session.ID)
	err := row.Scan(&sess.ID, &sess.UserID)
	if err == sql.ErrNoRows {
		return false, ErrNoSession
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *MySqlSessionRepository) DeleteSession(sessionid string) error {
	result, err := r.DB.Exec("DELETE FROM sessions WHERE sessionid = ?",
		sessionid)
	if err != nil {
		return err
	}

	affected, _ := result.RowsAffected()
	log.Println("destroyed sessions", affected)

	return nil
}
