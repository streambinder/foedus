package main

import (
	"log"
	"os"

	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/handlers"
	"github.com/streambinder/foedus/internal/middleware"
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

	app.Use(middleware.LangDetect())
	app.Static("/static", "./static")

	// public
	app.Get("/", handlers.Home)
	app.Post("/gift/claim", handlers.ClaimGift)

	// admin group (must be registered before the /:code catch-all)
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
	admin.Post("/guests/import", handlers.ImportGuestsCSV)
	admin.Get("/guests/:id/edit", handlers.EditGuestPage)
	admin.Post("/guests/:id", handlers.UpdateGuest)
	admin.Post("/guests/:id/delete", handlers.DeleteGuest)
	admin.Post("/guests/:id/confirm/:field", handlers.CycleConfirmed)
	admin.Post("/registry", handlers.AddRegistryItem)
	admin.Post("/registry/:id/delete", handlers.DeleteRegistryItem)
	admin.Post("/invitations", handlers.CreateInvitation)
	admin.Post("/invitations/:id/delete", handlers.DeleteInvitation)
	admin.Post("/polls", handlers.AddPoll)
	admin.Post("/polls/:id/delete", handlers.DeletePoll)

	// invitation public routes (catch-all, must be last)
	app.Get("/:code", handlers.ViewInvitation)
	app.Post("/:code/rsvp", handlers.UpdateInvitationRSVP)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(app.Listen(":" + port))
}
