package orders

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/KSerditov/Trading/api/exchange"
	"github.com/KSerditov/Trading/pkg/broker/user"
)

type OrdersRepositoryMySql struct {
	DSN string
	DB  *sql.DB
}

func (o *OrdersRepositoryMySql) SetupRepo() error {
	db, err := sql.Open("mysql", o.DSN)
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(10)

	err = db.Ping()
	if err != nil {
		return err
	}
	o.DB = db

	return nil
}

func NewOrdersRepositoryMySql(dsn string) (OrdersRepository, error) {
	repo := &OrdersRepositoryMySql{DSN: dsn}
	err := repo.SetupRepo()
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (o *OrdersRepositoryMySql) AddStatisticsEntity(e *exchange.OHLCV) (int64, error) {
	sql := "INSERT INTO stat(`time`, `interval`, `open`, `high`, `low`, `close`, `volume`, `ticker`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	res, err := o.DB.Exec(sql, e.Time, e.Interval, e.Open, e.High, e.Low, e.Close, e.Volume, e.Ticker)
	if err != nil {
		return -1, err
	}

	return res.LastInsertId()
}

func (o *OrdersRepositoryMySql) ChangeBalance(userid string, amount int32) (int32, error) {
	ctx := context.TODO()
	tx, txerr := o.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: false})
	if txerr != nil {
		return 0, txerr
	}
	defer tx.Rollback()

	u := user.User{}
	row := tx.QueryRow("SELECT BIN_TO_UUID(id) AS `id`, `username` AS `username` FROM users WHERE id = UUID_TO_BIN(?)", userid)
	err := row.Scan(&u.ID, &u.Username)
	if err != nil {
		return 0, err
	}

	var balance int32
	row2 := tx.QueryRow("SELECT `balance` FROM clients WHERE user_id = UUID_TO_BIN(?)", u.ID)
	err2 := row2.Scan(&balance)
	if err2 != nil || err2 == sql.ErrNoRows {
		return 0, err2
	}
	/* validate in handler if needed
	if balance+amount < 0 {
		return balance, errors.New("unsufficient balance to perform operation")
	}*/

	_, inserterr := tx.Exec(
		"REPLACE INTO clients (`user_id`, `balance`) VALUES (UUID_TO_BIN(?), ?)",
		u.ID,
		balance+amount,
	)
	if inserterr != nil {
		return balance, inserterr
	}
	commiterr := tx.Commit()
	if commiterr != nil {
		return balance, commiterr
	}

	return balance + amount, nil
}

func (o *OrdersRepositoryMySql) GetBalance(userid string) (int32, error) {
	var balance int32
	row := o.DB.QueryRow("SELECT clients.balance AS `balance` FROM `clients` INNER JOIN `users` ON clients.user_id = users.id WHERE users.id = UUID_TO_BIN(?)", userid)
	err := row.Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, user.ErrorUserNotFound
	}
	if err != nil {
		return 0, err
	}

	return balance, nil
}

func (o *OrdersRepositoryMySql) AddDeal(userid string, deal Deal) (int64, error) {
	fmt.Println("AddDeal checking user")
	// check user exists
	u := user.User{}
	row := o.DB.QueryRow("SELECT BIN_TO_UUID(id) AS `id`, `username` AS `username` FROM users WHERE id = UUID_TO_BIN(?);", userid)
	err := row.Scan(&u.ID, &u.Username)
	if err != nil {
		return -1, err
	}

	fmt.Println("AddDeal saving deal")
	// save deal
	var isBuy int8
	if strings.ToLower(deal.Type) == "buy" {
		isBuy = 1
	}
	sql := "INSERT INTO request (`id`, `user_id`, `ticker`, `volume`, `price`, `is_buy`) VALUES (?, UUID_TO_BIN(?), ?, ?, ?, ?);"
	res, errins := o.DB.Exec(sql, deal.Id, userid, deal.Ticker, deal.Volume, deal.Price, isBuy)
	if errins != nil {
		return -1, errins
	}

	fmt.Println("AddDeal done")
	return res.LastInsertId()
}

func (o *OrdersRepositoryMySql) DeleteDealById(id int64) error {
	sql := "DELETE FROM request WHERE id = ?"
	res, err := o.DB.Exec(sql, id)
	if err != nil {
		return err
	}

	aff, _ := res.RowsAffected()
	if aff <= 0 {
		return errors.New("no request with this id found")
	}

	return nil
}

func (o *OrdersRepositoryMySql) GetDealByUserAndId(userid string, dealid int64) (*Deal, error) {
	query := "SELECT `id`, `ticker`, `volume`, `price`, `is_buy` FROM request WHERE user_id = UUID_TO_BIN(?) AND id = ?"
	deal := &Deal{}
	row := o.DB.QueryRow(query, userid, dealid)
	err := row.Scan(&deal)
	if err == sql.ErrNoRows {
		return nil, ErrorDealNotFound
	}
	if err != nil {
		return nil, err
	}
	return deal, nil
}

func (o *OrdersRepositoryMySql) GetDealById(dealid int64) (*Deal, string, error) {
	query := "SELECT `id`, `ticker`, `volume`, `price`, `is_buy`, BIN_TO_UUID(user_id) AS `userid` FROM request WHERE id = ?"
	deal := &Deal{}
	var is_buy int8
	var userid string
	row := o.DB.QueryRow(query, dealid)
	err := row.Scan(&deal.Id, &deal.Ticker, &deal.Volume, &deal.Price, &is_buy, &userid)
	if err == sql.ErrNoRows {
		return nil, "", ErrorDealNotFound
	}
	if err != nil {
		return nil, "", err
	}
	if is_buy == 1 {
		deal.Type = "buy"
	} else {
		deal.Type = "sell"
	}
	return deal, userid, nil
}

func (o *OrdersRepositoryMySql) GetDealsByUserId(userid string) ([]Deal, error) {
	query := "SELECT `id`, `ticker`, `volume`, `price`, `is_buy` FROM request WHERE user_id = UUID_TO_BIN(?)"
	rows, err := o.DB.Query(query, userid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deals := make([]Deal, 0, 10)
	for rows.Next() {
		var deal Deal
		var isBuy bool
		err := rows.Scan(&deal.Id, &deal.Ticker, &deal.Volume, &deal.Price, &isBuy)
		if err != nil {
			return deals, err
		}
		if isBuy {
			deal.Type = "buy"
		} else {
			deal.Type = "sell"
		}
		deals = append(deals, deal)
	}
	if err = rows.Err(); err != nil {
		return deals, err
	}

	return deals, nil
}

func (o *OrdersRepositoryMySql) GetPositionsByUserId(userid string) ([]Position, error) {
	query := "SELECT `ticker`, `volume` FROM positions WHERE user_id = UUID_TO_BIN(?)"
	rows, err := o.DB.Query(query, userid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make([]Position, 0, 10)
	for rows.Next() {
		var position Position
		err := rows.Scan(&position.Ticker, &position.Volume)
		if err != nil {
			return positions, err
		}
		positions = append(positions, position)
	}
	if err = rows.Err(); err != nil {
		return positions, err
	}

	return positions, nil
}

func (o *OrdersRepositoryMySql) GetPositionByUserId(userid string, ticker string) (*Position, error) {
	query := "SELECT `volume` FROM positions WHERE user_id = UUID_TO_BIN(?) AND ticker = ?"
	row := o.DB.QueryRow(query, userid, ticker)
	position := &Position{
		Ticker: ticker,
		Volume: 0,
	}
	err := row.Scan(&position.Volume)
	if err == sql.ErrNoRows {
		return position, nil
	}
	if err != nil {
		return nil, err
	}

	return position, nil
}

func (o *OrdersRepositoryMySql) GetStatisticSince(since time.Time, ticker string) ([]Ohlcv, error) {
	query := "SELECT `time`, `open`, `high`, `low`, `close`, `volume` FROM stat WHERE time > ? AND ticker = ? ORDER BY time DESC"
	rows, err := o.DB.Query(query, since.Unix(), ticker)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ohlcvs := make([]Ohlcv, 0, 50)

	for rows.Next() {
		var ohlcv Ohlcv
		err := rows.Scan(&ohlcv.Time, &ohlcv.Open, &ohlcv.High, &ohlcv.Low, &ohlcv.Close, &ohlcv.Volume)
		if err != nil {
			return ohlcvs, err
		}
		ohlcvs = append(ohlcvs, ohlcv)
	}
	if err = rows.Err(); err != nil {
		return ohlcvs, err
	}
	return ohlcvs, nil
}

func (o *OrdersRepositoryMySql) ChangePosition(userid string, ticker string, volumeChange int32) (*Position, error) {
	fmt.Println("ChangePosition")
	ctx := context.TODO()
	tx, txerr := o.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: false})
	if txerr != nil {
		return nil, txerr
	}
	defer tx.Rollback()

	var positionid int64
	position := &Position{
		Ticker: ticker,
		Volume: 0,
	}
	query := "SELECT `id`, `volume` FROM positions WHERE user_id = UUID_TO_BIN(?) AND ticker = ?"
	row := tx.QueryRow(query, userid, ticker)
	err := row.Scan(&positionid, &position.Volume)
	fmt.Println("ChangePosition scan:")
	fmt.Println(positionid)
	fmt.Println(position)
	if err == sql.ErrNoRows {
		insert := "INSERT INTO `positions` (`user_id`,`ticker`,`volume`) VALUES (UUID_TO_BIN(?), ?, ?)"
		r, err2 := tx.Exec(insert, userid, ticker, 0)
		fmt.Println("exec done")
		if err2 != nil {
			return nil, err2
		}
		fmt.Println("go to lastinsertedid")
		positionid, err2 = r.LastInsertId()
		if err2 != nil {
			return nil, err2
		}
		fmt.Println("inserted")
	} else if err != nil {
		fmt.Println("error?")
		fmt.Println(err)
		return nil, err
	}

	position.Volume += volumeChange
	fmt.Println("postiion volume")
	update := "UPDATE `positions` SET `volume` = ? WHERE `id` = ?"
	_, err2 := tx.Exec(update, position.Volume, positionid)
	if err2 != nil {
		return nil, err2
	}

	commiterr := tx.Commit()
	if commiterr != nil {
		return nil, commiterr
	}

	return position, nil
}
