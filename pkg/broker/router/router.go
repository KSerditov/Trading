package router

import (
	"html/template"
	"net/http"
	"os"

	"github.com/KSerditov/Trading/pkg/broker/custlog"
	"github.com/KSerditov/Trading/pkg/broker/exchclient"
	"github.com/KSerditov/Trading/pkg/broker/handlers"
	"github.com/KSerditov/Trading/pkg/broker/middleware"
	"github.com/KSerditov/Trading/pkg/broker/orders"
	"github.com/KSerditov/Trading/pkg/broker/session"
	"github.com/KSerditov/Trading/pkg/broker/user"

	"github.com/gorilla/mux"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type BrokerApp struct {
	Router      *http.Handler
	SessionRepo *session.SessionRepository
	UserRepo    *user.UserRepository
	OrdersRepo  *orders.OrdersRepository
	Logger      *custlog.Logger
}

func (a *BrokerApp) Initialize(sessRepo *session.SessionRepository, userRepo *user.UserRepository, ordersRepo *orders.OrdersRepository) {
	a.Logger = &custlog.Logger{
		Zap:   getBaseLogger(),
		Level: 0,
	}
	a.SessionRepo = sessRepo
	a.UserRepo = userRepo
	a.OrdersRepo = ordersRepo

	sm := &session.JWTSessionManager{
		Secret:      []byte("supersecretkey"),
		SessionRepo: *a.SessionRepo,
	}

	AuthMiddlware := middleware.AuthHandler{
		SessMgr:  sm,
		UserRepo: *a.UserRepo,
	}

	ExchangeClient := &exchclient.OrderExchClientGRPC{
		ExchServerAddress: "127.0.0.1:8082",
		BrokerID:          123,
	}
	errexch := ExchangeClient.Init()
	if errexch != nil {
		a.Logger.Zap.Fatal("grpc client initialization failure")
	}

	OrderHandlers := &handlers.OrderHandlers{
		SessMgr:         sm,
		OrdersRepo:      *a.OrdersRepo,
		ExchClient:      ExchangeClient,
		ClientID:        11,
		HistoryDepthMin: 15,
	}

	UserHandlers := &handlers.UserHandlers{
		SessMgr:    sm,
		UserRepo:   *a.UserRepo,
		OrdersRepo: *a.OrdersRepo,
	}

	templates := template.Must(template.ParseGlob("c:/repositories/trading/web/templates/*"))

	UserClientHandlers := &handlers.UserClientHandler{
		BrokerBaseUrl: "http://127.0.0.1:8080",
		Tmpl:          templates,
		Logger:        a.Logger.Zap.Sugar(),

		UserAPI:   UserHandlers,
		OrdersAPI: OrderHandlers,
	}

	/*r.Path("/").Handler(http.FileServer(http.Dir("./web/")))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))
	*/

	r := mux.NewRouter()

	r1 := r.PathPrefix("/api/v1/").Subrouter()
	r1.HandleFunc("/register", UserHandlers.Register)
	r1.HandleFunc("/login", UserHandlers.Login)
	r1.HandleFunc("/logout", UserHandlers.Logout)
	r1.HandleFunc("/deal", OrderHandlers.CreateDealHr)
	r1.HandleFunc("/cancel", OrderHandlers.CancelDealHr)
	r1.HandleFunc("/status", OrderHandlers.GetStatus)
	r1.HandleFunc("/history", OrderHandlers.GetHistory)
	r1.Use(AuthMiddlware.Auth)

	r.HandleFunc("/login", UserClientHandlers.Login)
	r.HandleFunc("/logout", UserClientHandlers.Logout)
	r.HandleFunc("/deal", UserClientHandlers.Deal)
	r.HandleFunc("/positions", UserClientHandlers.Positions)
	r.HandleFunc("/history", UserClientHandlers.History)
	r.HandleFunc("/", UserClientHandlers.Index)
	r.Use(AuthMiddlware.Auth)

	router := middleware.Slasher(r)
	router = middleware.AccessLog(router)
	router = a.Logger.SetupLogger(router)
	router = a.Logger.SetupReqID(router)
	router = middleware.Panic(a.Logger.Zap.Sugar(), router)

	a.Router = &router
}

func getBaseLogger() *zap.Logger {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	consoleEncoder := zapcore.NewConsoleEncoder(config)
	logFile, _ := os.OpenFile("log/log.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	writer := zapcore.AddSync(logFile)
	defaultLogLevel := zapcore.DebugLevel
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), defaultLogLevel),
	)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger
}
