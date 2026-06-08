package router

import (
	"auction/internal/handler"
	"auction/internal/middleware"
	"auction/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handlers struct {
	Auth            *handler.AuthHandler
	User            *handler.UserHandler
	Category        *handler.CategoryHandler
	Product         *handler.ProductHandler
	LiveRoom        *handler.LiveRoomHandler
	AuctionSession  *handler.AuctionSessionHandler
	Bid             *handler.BidHandler
	Order           *handler.OrderHandler
	PaymentRecord   *handler.PaymentRecordHandler
	Notification    *handler.NotificationHandler
	AuctionLog      *handler.AuctionLogHandler
	Seller          *handler.SellerHandler
	Buyer           *handler.BuyerHandler
	Comment         *handler.CommentHandler
	Like            *handler.LikeHandler
	WS              *handler.WSHandler
}

func NewRouter(h *Handlers, logger *zap.Logger, jwtSecret string, db *gorm.DB) *gin.Engine {
	r := gin.New()

	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS())

	r.Static("/static/upload", "./static/upload")

	r.GET("/api/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"})
	})

	// Server time for client clock calibration (Unix ms)
	r.GET("/api/server-time", func(c *gin.Context) {
		response.Success(c, gin.H{"server_time": time.Now().UnixMilli()})
	})

	api := r.Group("/api")

	auth := api.Group("/auth")
	{
		auth.POST("/register", h.Auth.Register)
		auth.POST("/login", h.Auth.Login)
	}

	api.GET("/ws", h.WS.Connect)

	// Public read routes
	api.GET("/categories", h.Category.List)
	api.GET("/categories/:id", h.Category.GetByID)
	api.GET("/products", h.Product.List)
	api.GET("/products/:id", h.Product.GetByID)
	api.GET("/live-rooms", h.LiveRoom.List)
	api.GET("/live-rooms/:id", h.LiveRoom.GetByID)
	api.GET("/live-rooms/:id/comments", h.Comment.List)
	api.GET("/live-rooms/:id/likes", h.Like.Total)
	api.GET("/live-rooms/:id/auction", h.Buyer.GetCurrentAuction)
		api.GET("/live-rooms/:id/products", h.Buyer.ListRoomProducts)
	api.GET("/auction-sessions", h.AuctionSession.List)
	api.GET("/auction-sessions/:id/ranking", h.Buyer.GetBidRanking)
	api.GET("/auction-sessions/:id", h.AuctionSession.GetByID)
	api.GET("/auction-sessions/:id/online", h.AuctionSession.GetOnlineCount)
	api.GET("/bids", h.Bid.ListByAuction)

	// JWT required
	authRequired := api.Group("")
	authRequired.Use(middleware.Auth(jwtSecret, db))
	{
		authRequired.GET("/users/:id", h.User.GetByID)
		authRequired.PUT("/users/:id", h.User.Update)
		authRequired.POST("/upload/image", h.Seller.UploadImage)

		authRequired.POST("/bids", h.Bid.Create)
		authRequired.GET("/bids/mine", h.Buyer.ListMyBids)
	authRequired.GET("/bids/history", h.Buyer.BidHistory)
		authRequired.POST("/live-rooms/:id/comments", h.Comment.Send)
		authRequired.POST("/live-rooms/:id/like", h.Like.Send)

		authRequired.GET("/orders", h.Order.ListMyOrders)
		authRequired.GET("/orders/:id", h.Order.GetByID)
		authRequired.PUT("/orders/:id/address", h.Order.ConfirmAddress)
		authRequired.POST("/orders/:id/pay", h.Order.Pay)
		authRequired.POST("/orders/:id/confirm", h.Order.ConfirmReceipt)

		authRequired.GET("/payments", h.PaymentRecord.List)
		authRequired.GET("/payments/:id", h.PaymentRecord.GetByID)

		authRequired.GET("/notifications", h.Notification.ListByUser)
		authRequired.PUT("/notifications/:id/read", h.Notification.MarkRead)

		authRequired.GET("/auction-logs", h.AuctionLog.ListByAuction)
	}

	// Seller routes (JWT + seller/admin role required)
	seller := api.Group("/seller")
	seller.Use(middleware.Auth(jwtSecret, db))
	seller.Use(middleware.SellerOnly())
	{
		seller.POST("/upload/image", h.Seller.UploadImage)

		products := seller.Group("/products")
		{
			products.POST("", h.Seller.CreateProduct)
			products.PUT("/:id", h.Seller.UpdateProduct)
			products.GET("", h.Seller.ListProducts)
			products.GET("/:id", h.Seller.GetProduct)
			products.DELETE("/:id", h.Seller.DeleteProduct)
		}

		// Live room management
		rooms := seller.Group("/live-rooms")
		{
			rooms.POST("", h.LiveRoom.Create)
			rooms.PUT("/:id", h.LiveRoom.Update)
			rooms.GET("", h.LiveRoom.ListMyRooms)
			rooms.POST("/:id/start", h.LiveRoom.StartLive)
			rooms.POST("/:id/end", h.LiveRoom.EndLive)
		}

		sessions := seller.Group("/auction-sessions")
		{
			sessions.POST("", h.Seller.CreateAuctionSession)
			sessions.PUT("/:id", h.Seller.UpdateAuctionSession)
			sessions.GET("", h.Seller.ListAuctionSessions)
			sessions.POST("/:id/start", h.Seller.StartAuction)
			sessions.POST("/:id/cancel", h.Seller.CancelAuction)
		}

		seller.GET("/orders", h.Order.ListSellerOrders)
		seller.POST("/orders/:id/ship", h.Order.ShipOrder)
			seller.GET("/dashboard", h.Seller.Dashboard)
		}

	return r
}
