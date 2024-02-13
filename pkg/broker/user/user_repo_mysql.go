package user

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"github.com/google/uuid"
)

type MySqlUserRepository struct {
	DSN string
	DB  *sql.DB
}

// not sure how to cover this with test
// probably should be put outside of repository implementation
func (r *MySqlUserRepository) SetupRepo() error {
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

func NewMySqlUserRepository(dsn string) (UserRepository, error) {
	repo := &MySqlUserRepository{DSN: dsn}
	err := repo.SetupRepo()
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *MySqlUserRepository) AddUser(username string, password string) (User, error) {
	var exists int8
	ctx := context.TODO()

	tx, txerr := r.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: false})
	if txerr != nil {
		return User{}, txerr
	}
	defer tx.Rollback()

	row := tx.QueryRow("SELECT EXISTS(SELECT * FROM users WHERE username = ?) AS 'exists'", username)
	err := row.Scan(&exists)
	if err != nil {
		return User{}, err
	}
	if exists == 1 {
		return User{}, ErrorUserAlreadyExists
	}

	md5 := md5.Sum([]byte(password))

	u := User{
		Username:     username,
		PasswordHash: hex.EncodeToString(md5[:]),
		ID:           uuid.New().String(),
	}

	_, inserterr := tx.Exec(
		`INSERT INTO users (id, username, hash) VALUES (UUID_TO_BIN(?), ?, ?)`,
		u.ID,
		u.Username,
		u.PasswordHash,
	)
	if inserterr != nil {
		return User{}, inserterr
	}

	commiterr := tx.Commit()
	if commiterr != nil {
		return User{}, commiterr
	}

	return r.GetUser(username)
}

func (r *MySqlUserRepository) GetUser(username string) (User, error) {
	fmt.Printf("GetUser %v\n", username)
	u := &User{}
	row := r.DB.QueryRow("SELECT BIN_TO_UUID(id), username, hash FROM users WHERE username = ?", username)
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash)
	if err == sql.ErrNoRows {
		fmt.Println("HERE1!")
		return *u, ErrorUserNotFound
	}
	if err != nil {
		return *u, err
	}
	return *u, nil
}

func (r *MySqlUserRepository) GetUserById(userid string) (User, error) {
	fmt.Printf("GetUserById %v\n", userid)
	u := &User{}
	row := r.DB.QueryRow("SELECT * FROM users WHERE id = UUID_TO_BIN(?)", userid)
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash)
	if err == sql.ErrNoRows {
		fmt.Println("HERE2!")
		return *u, ErrorUserNotFound
	}
	if err != nil {
		return *u, err
	}
	return *u, nil
}

func (r *MySqlUserRepository) ValidatePassword(user User, password string) (bool, error) {
	md5 := md5.Sum([]byte(password))
	if hex.EncodeToString(md5[:]) == user.PasswordHash {
		return true, nil
	}
	return false, nil
}
