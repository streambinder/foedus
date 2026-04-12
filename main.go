//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/handlers"
	"github.com/streambinder/foedus/internal/middleware"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "foedus.db"
	}
	database.Init(dsn)

	openrouterKey := os.Getenv("OPENROUTER_API_KEY")
	openrouterModel := os.Getenv("OPENROUTER_MODEL")
	if openrouterModel == "" {
		openrouterModel = "meta-llama/llama-3.3-70b-instruct:free"
	}
	handlers.InitChat(openrouterKey, openrouterModel)

	app := fiber.New(fiber.Config{
		BodyLimit: 16 * 1024 * 1024,
	})

	app.Use(middleware.LangDetect())
	app.Static("/static", "./static")

	// public
	app.Get("/", handlers.Home)
	app.Get("/og-image", handlers.OGImage)
	app.Post("/gift/claim", handlers.ClaimGift)
	app.Post("/chat", handlers.ChatStream)

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
	admin.Get("/gifts/:id/edit", handlers.EditGiftPage)
	admin.Post("/gifts/:id", handlers.UpdateGift)
	admin.Post("/gifts/:id/delete", handlers.DeleteGift)
	admin.Post("/invitations", handlers.CreateInvitation)
	admin.Post("/invitations/:id/delete", handlers.DeleteInvitation)
	admin.Post("/polls", handlers.AddPoll)
	admin.Post("/polls/:id/delete", handlers.DeletePoll)

	// invitation public routes (catch-all, must be last)
	app.Get("/:code", handlers.ViewInvitation)
	app.Post("/:code/viewed", handlers.MarkInvitationViewed)
	app.Post("/:code/rsvp", handlers.UpdateInvitationRSVP)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(app.Listen(":" + port))
}
