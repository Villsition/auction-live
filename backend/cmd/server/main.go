package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auction/internal/config"
	"auction/internal/handler"
	redisPkg "auction/pkg/redis"
	"auction/internal/repository"
	"auction/internal/router"
	"auction/internal/scheduler"
	"auction/internal/service"
	"auction/internal/ws"
	"auction/pkg/upload"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func gormLogLevel(lvl string) logger.LogLevel {
	switch lvl {
	case "info":
		return logger.Info
	case "warn":
		return logger.Warn
	case "error":
		return logger.Error
	default:
		return logger.Info
	}
}

func main() {
	cfgPath := "config/config.yaml"
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		cfgPath = v
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	zapLog := newZapLogger(cfg.Log.Level, cfg.Log.File)
	defer zapLog.Sync()

	db, err := gorm.Open(mysql.Open(cfg.DB.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel(cfg.Log.Level)),
	})
	if err != nil {
		zapLog.Fatal("failed to connect database", zap.Error(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		zapLog.Fatal("failed to get sql.DB", zap.Error(err))
	}
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)

	// Ensure performance indexes exist (ignore error if already present)
	if err := db.Exec("CREATE INDEX idx_bids_user_valid ON bids(user_id, is_valid)").Error; err != nil {
		zapLog.Debug("index may already exist", zap.String("index", "idx_bids_user_valid"), zap.Error(err))
	}

	zapLog.Info("database connected", zap.String("host", cfg.DB.Host))

	rdb, err := redisPkg.NewClient(cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB, cfg.Redis.PoolSize)
	if err != nil {
		zapLog.Fatal("failed to connect redis", zap.Error(err))
	}
	zapLog.Info("redis connected", zap.String("addr", cfg.Redis.Addr()))

	rdbRead := rdb
	if cfg.Redis.ReadAddr() != cfg.Redis.Addr() {
		rdbRead, err = redisPkg.NewClient(cfg.Redis.ReadAddr(), cfg.Redis.Password, cfg.Redis.DB, cfg.Redis.PoolSize)
		if err != nil {
			zapLog.Fatal("failed to connect redis read replica", zap.Error(err))
		}
		zapLog.Info("redis read replica connected", zap.String("addr", cfg.Redis.ReadAddr()))
	}

	hub := ws.NewHub(rdb, rdbRead, zapLog)
	go hub.Run()

	userRepo := repository.NewUserRepo(db)
	categoryRepo := repository.NewCategoryRepo(db)
	productRepo := repository.NewProductRepo(db)
	liveRoomRepo := repository.NewLiveRoomRepo(db)
	auctionSessionRepo := repository.NewAuctionSessionRepo(db, rdb, rdbRead)
	bidRepo := repository.NewBidRepo(db, rdb, rdbRead)
	orderRepo := repository.NewOrderRepo(db)
	paymentRecordRepo := repository.NewPaymentRecordRepo(db)
	notificationRepo := repository.NewNotificationRepo(db)
	commentRepo := repository.NewCommentRepo(db, rdb, rdbRead)
	auctionLogRepo := repository.NewAuctionLogRepo(db)

	userSvc := service.NewUserSvc(userRepo)
	commentSvc := service.NewCommentSvc(commentRepo, userRepo, hub)
	authSvc := service.NewAuthSvc(userRepo, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	categorySvc := service.NewCategorySvc(categoryRepo)
	productSvc := service.NewProductSvc(productRepo)
	liveRoomSvc := service.NewLiveRoomSvc(liveRoomRepo, rdb)
	auctionSessionSvc := service.NewAuctionSessionSvc(auctionSessionRepo)
	orderSvc := service.NewOrderSvc(orderRepo)
	paymentRecordSvc := service.NewPaymentRecordSvc(paymentRecordRepo)
	notifSvc := service.NewNotificationSvc(notificationRepo, hub)
	bidSvc := service.NewBidSvc(bidRepo, notifSvc, hub)
	auctionLogSvc := service.NewAuctionLogSvc(auctionLogRepo)

	uploader := upload.NewUploader("./static/upload", "/static/upload", 10)

	handlers := &router.Handlers{
		Auth:            handler.NewAuthHandler(authSvc, uploader),
		User:            handler.NewUserHandler(userSvc),
		Category:        handler.NewCategoryHandler(categorySvc),
		Product:         handler.NewProductHandler(productSvc),
		LiveRoom:        handler.NewLiveRoomHandler(liveRoomSvc, rdb, commentSvc, auctionSessionSvc, hub),
		AuctionSession:  handler.NewAuctionSessionHandler(auctionSessionSvc, hub),
		Bid:             handler.NewBidHandler(bidSvc),
		Order:           handler.NewOrderHandler(orderSvc),
		PaymentRecord:   handler.NewPaymentRecordHandler(paymentRecordSvc),
		Notification:    handler.NewNotificationHandler(notifSvc),
		AuctionLog:      handler.NewAuctionLogHandler(auctionLogSvc),
		Seller:          handler.NewSellerHandler(productSvc, auctionSessionSvc, bidSvc, orderSvc, uploader, hub),
		Buyer:           handler.NewBuyerHandler(userRepo, liveRoomSvc, auctionSessionSvc, bidSvc, productSvc, db),
		Comment:         handler.NewCommentHandler(commentSvc),
		Like:            handler.NewLikeHandler(rdb, hub),
		WS:              handler.NewWSHandler(hub, cfg.JWT.Secret, zapLog),
		Health:          handler.NewHealthHandler(db, rdb),
	}

	watcher := scheduler.NewAuctionWatcher(rdb, auctionSessionRepo, bidRepo, orderRepo, userRepo, notifSvc, hub, zapLog)
	watcher.Start()

	likeFlusher := scheduler.NewLikeFlusher(rdb, db, zapLog)
	likeFlusher.Start()

	r := router.NewRouter(handlers, zapLog, cfg.JWT.Secret, db)

	ginMode := cfg.Server.Mode
	if ginMode == "" {
		ginMode = "release"
	}
	gin.SetMode(ginMode)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Start server in background
	go func() {
		zapLog.Info("server starting", zap.String("addr", addr), zap.String("mode", ginMode))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLog.Fatal("server failed", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	zapLog.Info("shutdown signal received", zap.String("signal", sig.String()))

	// Stop background schedulers first
	watcher.Stop()
	likeFlusher.Stop()

	// Graceful HTTP shutdown (wait up to 30s for in-flight requests)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zapLog.Error("server forced to shutdown", zap.Error(err))
	}

	// Close DB and Redis
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}
	rdb.Close()

	zapLog.Info("server exited gracefully")
	zapLog.Sync()
}

func newZapLogger(level, file string) *zap.Logger {
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = zapcore.DebugLevel
	}
	if file != "" {
		return newFileLogger(lvl, file)
	}
	return newConsoleLogger(lvl)
}

func newConsoleLogger(lvl zapcore.Level) *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapLog, _ := cfg.Build()
	return zapLog
}

func newFileLogger(lvl zapcore.Level, path string) *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.OutputPaths = []string{path}
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapLog, _ := cfg.Build()
	return zapLog
}
