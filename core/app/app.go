package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"hei-gin/config"
	"hei-gin/core/auth"
	"hei-gin/core/captcha"
	"hei-gin/core/db"
	"hei-gin/core/middleware"
	"hei-gin/core/utils"
)

func Run() {
	// 1. Load config
	if err := config.Load("config.yaml"); err != nil {
		log.Fatalf("[APP] Failed to load config: %v", err)
	}

	// 2. Init DB
	if err := db.InitEnt(); err != nil {
		log.Fatalf("[APP] Failed to init database: %v", err)
	}

	// 3. Init Redis
	if err := db.InitRedis(); err != nil {
		log.Fatalf("[APP] Failed to init Redis: %v", err)
	}

	// 4. SM2 Init
	utils.Init(config.C.SM2.PrivateKey, config.C.SM2.PublicKey)

	// 5. Auth tool init
	auth.Init(config.C.JWT.ExpireSeconds, config.C.JWT.TokenName)
	auth.NewHeiClientAuthTool().Init(config.C.JWT.ExpireSeconds, config.C.JWT.TokenName)

	// 6. Register permission interface
	auth.RegisterInterface(&auth.HeiPermissionInterfaceImpl{})

	// 7. Init captcha
	captcha.BCaptcha.Init(db.Redis)
	captcha.CCaptcha.Init(db.Redis)

	// 8. Init auth login user provider

	// 9. Create Gin engine
	r := gin.Default()

	// 10. Global middleware
	r.Use(middleware.Trace())
	r.Use(middleware.AuthCheck())
	r.Use(middleware.Recovery())

	// 11. CORS
	r.Use(middleware.CORS())

	// 12. Setup routes
	SetupRouters(r)

	// 13. Run permission scan
	auth.RunPermissionScan()

	// 14. Start HTTP server with graceful shutdown
	addr := fmt.Sprintf("%s:%d", config.C.App.Host, config.C.App.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("[APP] Server started on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[APP] Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[APP] Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[APP] Server forced to shutdown: %v", err)
	}

	db.Close()
	db.CloseRedis()
	log.Println("[APP] Server exited")
}
