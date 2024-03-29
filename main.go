package main

import (
	"awesomeProject/api"
	"awesomeProject/client"
	_ "awesomeProject/docs" // docs is generated by Swag CLI, you have to import it.
	service "awesomeProject/external"
	"context"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	_ "github.com/swaggo/http-swagger"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// @title API Title
// @version 1.0
// @description This is a sample external for your API.
// @termsOfService http://example.com/terms/
// @contact name API Support email support@example.com
// @license name Apache 2.0 url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @BasePath /
// @schemes http https
// swagger: "2.0"
func main() {
	err := godotenv.Load("./config.env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	viper.AutomaticEnv()

	viper.SetDefault("SERVICE_LIMIT", 5)
	viper.SetDefault("SERVICE_PERIOD", 2)

	serviceLimit := viper.GetInt("SERVICE_LIMIT")
	servicePeriod := viper.GetInt("SERVICE_PERIOD")

	periodDuration := time.Duration(servicePeriod) * time.Second
	blockDuration := time.Duration(servicePeriod) * 2 * time.Second

	mockService := service.NewMockService(uint64(serviceLimit), periodDuration, blockDuration)
	clientService := client.NewClient(mockService)

	apiHandler := api.NewAPI(clientService)

	mux := http.NewServeMux()
	apiHandler.SetupRoutes(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Create a channel to listen for interrupt or termination signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	<-quit
	log.Println("Shutting down external...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")

}
