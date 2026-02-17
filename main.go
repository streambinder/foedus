package main

import (
	"log"
	"os"

	"github.com/dpucci/foedus/internal/database"
	"github.com/dpucci/foedus/internal/handlers"
	"github.com/dpucci/foedus/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/csrf"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "foedus.db"
	}
	database.Init(dsn)

	app := fiber.New()

	// public
	app.Get("/", handlers.Home)

	// admin group
	admin := app.Group("/dashboard", middleware.BasicAuth())

	// csrf for all dashboard routes — token extracted from form field "_csrf"
	admin.Use(csrf.New(csrf.Config{
		KeyLookup:      "form:_csrf",
		ContextKey:     "csrf",
		CookieSameSite: "Strict",
	}))

	admin.Get("/", handlers.DashboardIndex)
	admin.Post("/settings", handlers.SaveSettings)
	admin.Post("/guests", handlers.AddGuest)
	admin.Get("/guests/:id/edit", handlers.EditGuestPage)
	admin.Post("/guests/:id", handlers.UpdateGuest)
	admin.Post("/guests/:id/delete", handlers.DeleteGuest)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(app.Listen(":" + port))
}
