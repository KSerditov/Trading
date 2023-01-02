package main

import (
	"net/http"

	"github.com/KSerditov/Trading/api/exchange"
	"github.com/KSerditov/Trading/pkg/broker/orders"
	"github.com/KSerditov/Trading/pkg/broker/router"
	"github.com/KSerditov/Trading/pkg/broker/session"
	"github.com/KSerditov/Trading/pkg/broker/user"
)

/* TBD FOR BROKER

1. Write web api schema based on swagger

	GET /api/v1/status - returns balance + list of positions
	POST /api/v1/deal - sends new order request
	POST /api/v1/cancel - cancels order request
	GET /api/v1/history?ticker=SPFB.RTS - returns history for ticker
	POST /api/v1/login
	POST /api/v1/register - username + password

2. Initialize DB instance and store everything there

4. Add logging

5. Write tests

6. Missing cookie save

7. No way to obtaine missed data due to broker/connection failure

8. Check amounts, balance int overflows

9. gRPC connections reuse/pool?
*/

func main() {

	app := router.BrokerApp{}

	s, err := session.NewMySqlSessionRepository("root@tcp(localhost:3306)/broker?&charset=utf8")
	if err != nil {
		app.Logger.Zap.Sugar().Errorw("failed to initialize mysql repository for session",
			"error", err.Error(),
		)
	}

	u, mysqlerr := user.NewMySqlUserRepository("root@tcp(localhost:3306)/broker?&charset=utf8")
	if mysqlerr != nil {
		app.Logger.Zap.Sugar().Errorw("failed to initialize mysql repository for user",
			"error", mysqlerr.Error(),
		)
	}

	o, oerr := orders.NewOrdersRepositoryMySql("root@tcp(localhost:3306)/broker?&charset=utf8")
	if oerr != nil {
		app.Logger.Zap.Sugar().Errorw("failed to initialize mysql repository for user",
			"error", oerr.Error(),
		)
	}

	ol := orders.OrdersListener{
		ExchServerAddress: "127.0.0.1:8082",
		BrokerID: &exchange.BrokerID{
			ID: 123,
		},
		OrdersRepository: o,
	}
	ol.Start()

	app.Initialize(&s, &u, &o)

	addr := ":8080"
	app.Logger.Zap.Sugar().Infow("starting server",
		"type", "START",
		"addr", addr,
	)
	http.ListenAndServe(addr, *app.Router)
}
