package router

import (
	"net/http"
	"os"

	"github.com/KSerditov/Trading/pkg/broker/custlog"
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

	OrderHandlers := &handlers.OrderHandlers{
		SessMgr:           sm,
		OrdersRepo:        *a.OrdersRepo,
		ExchServerAddress: "127.0.0.1:8082",
		BrokerID:          123,
		ClientID:          11,
		HistoryDepthMin:   15,
	}

	UserHandlers := &handlers.UserHandlers{
		SessMgr:    sm,
		UserRepo:   *a.UserRepo,
		OrdersRepo: *a.OrdersRepo,
	}

	/*r.Path("/").Handler(http.FileServer(http.Dir("./web/")))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))

	r1 := r.PathPrefix("/api/posts").Subrouter()
	r1.Use(middleware.AddJsonContent)
	r1get := r1.Methods("GET").Subrouter()
	r1post := r1.Methods("POST").Subrouter()

	r1get.HandleFunc("/{category}", ContentHandlers.CategoryPosts)
	r1get.HandleFunc("", ContentHandlers.Posts)

	r1post.HandleFunc("", ContentHandlers.AddPost)
	r1post.Use(AuthMiddlware.Auth)

	r2 := r.PathPrefix("/api/post").Subrouter()
	r2.Use(middleware.AddJsonContent)

	r2get := r2.Methods("GET").Subrouter()
	r2post := r2.Methods("POST").Subrouter()
	r2del := r2.Methods("DELETE").Subrouter()

	r2auth := r2.PathPrefix("/{postid}/").Methods("GET").Subrouter()
	r2auth.HandleFunc("/upvote", ContentHandlers.UpvotePost)
	r2auth.HandleFunc("/downvote", ContentHandlers.DownvotePost)
	r2auth.HandleFunc("/unvote", ContentHandlers.UnvotePost)
	r2auth.Use(AuthMiddlware.Auth)

	r2get.HandleFunc("/{postid}", ContentHandlers.Post)

	r2post.HandleFunc("/{postid}", ContentHandlers.AddComment)
	r2post.Use(AuthMiddlware.Auth)

	r2del.HandleFunc("/{postid}/{commentid}", ContentHandlers.DeleteComment)
	r2del.HandleFunc("/{postid}", ContentHandlers.DeletePost)
	r2del.Use(AuthMiddlware.Auth)*/

	r := mux.NewRouter()
	r.HandleFunc("/login", UserHandlers.Login)
	r.HandleFunc("/register", UserHandlers.Register)

	r1 := r.PathPrefix("/api/v1/").Subrouter()
	r1.HandleFunc("/deal", OrderHandlers.CreateDeal)
	r1.HandleFunc("/cancel", OrderHandlers.CancelDeal)
	r1.HandleFunc("/status", OrderHandlers.GetStatus)
	r1.HandleFunc("/history", OrderHandlers.GetHistory)
	r1.Use(AuthMiddlware.Auth)

	//api := r.PathPrefix("/api").Subrouter()
	//api.HandleFunc("/login", UserHandlers.Login)
	//api.HandleFunc("/register", UserHandlers.Register)
	//api.HandleFunc("/user/{username}", ContentHandlers.UserPosts)
	//api.Use(middleware.AddJsonContent)

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
