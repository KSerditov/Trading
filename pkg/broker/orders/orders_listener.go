package orders

import (
	"context"
	"os"

	"github.com/KSerditov/Trading/api/exchange"
	"github.com/KSerditov/Trading/pkg/broker/custlog"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OrdersListener struct {
	ExchServerAddress string
	Logger            *custlog.Logger
	BrokerID          *exchange.BrokerID
	OrdersRepository  OrdersRepository
}

func (o *OrdersListener) Start() error {
	o.Logger = &custlog.Logger{
		Zap:   getBaseLogger(),
		Level: 0,
	}

	o.Logger.Zap.Info("Setting up gRPC connection to Exchange server...")
	grcpConn, err := grpc.Dial(
		o.ExchServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		o.Logger.Zap.Fatal("Error initializing gRPC connection to exchange", zap.Error(err))
	}

	exch := exchange.NewExchangeClient(grcpConn)

	ctx := context.Background()
	statistics, err := exch.Statistic(ctx, o.BrokerID)
	if err != nil {
		o.Logger.Zap.Fatal("Error initializing gRPC connection to exchange", zap.Error(err))
	}

	results, err := exch.Results(ctx, o.BrokerID)
	if err != nil {
		o.Logger.Zap.Fatal("error initializing gRPC connection to exchange", zap.Error(err))
	}

	go func() {
		for {
			resp, err := statistics.Recv()
			if err != nil {
				o.Logger.Zap.Error("can't receive from grpc statistics stream", zap.String("error", err.Error()))
				return
			}
			_, errdb := o.OrdersRepository.AddStatisticsEntity(resp)
			if errdb != nil {
				o.Logger.Zap.Error("failed to save interval statistics to db", zap.String("error", err.Error()))
			}
		}
	}()

	go func() {
		for {
			result, err := results.Recv()
			if err != nil {
				o.Logger.Zap.Error("can't receive from grpc statistics stream", zap.String("error", err.Error()))
				continue
			}

			o.Logger.Zap.Sugar().Debugw("result received from exchange", "result", result)
			_, userid, err := o.OrdersRepository.GetDealById(result.ID)
			if err != nil {
				o.Logger.Zap.Sugar().Errorw("unable to find local details for deal received for exchange",
					"result", result,
					"error", err,
				)
				continue
			}

			var balanceChange int32
			var volumeChange int32

			if result.Volume > 0 {
				balanceChange = -result.Volume * int32(result.Price)
				volumeChange = result.Volume
			} else {
				balanceChange = result.Volume * int32(result.Price)
				volumeChange = -result.Volume
			}

			_, err1 := o.OrdersRepository.ChangeBalance(userid, balanceChange)
			if err1 != nil {
				o.Logger.Zap.Sugar().Errorw("failed to change balance",
					"result", result,
					"userid", userid,
					"proposed_change", balanceChange,
					"error", err,
				)
			}
			_, err2 := o.OrdersRepository.ChangePosition(userid, result.Ticker, volumeChange)
			if err2 != nil {
				o.Logger.Zap.Sugar().Errorw("failed to change position",
					"result", result,
					"userid", userid,
					"proposed_change", volumeChange,
					"error", err,
				)
			}

			if !result.Partial {
				delerr := o.OrdersRepository.DeleteDealById(result.ID)
				if delerr != nil {
					o.Logger.Zap.Sugar().Errorw("failed to delete completed order",
						"result", result,
						"userid", userid,
						"proposed_change", volumeChange,
						"error", err,
					)
				}
			}
		}
	}()

	return nil
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
