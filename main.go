//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

package main

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/streambinder/foedus/internal/database"
	"github.com/streambinder/foedus/internal/handlers"
	"github.com/streambinder/foedus/internal/middleware"
	"github.com/streambinder/foedus/internal/observability"
	"github.com/streambinder/foedus/internal/spotify"
)

func main() {
	observability.Init()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "foedus.db"
	}
	slog.Info("initializing application", "dsn", dsn)
	database.Init(dsn)

	openrouterKey := os.Getenv("OPENROUTER_API_KEY")
	openrouterModel := os.Getenv("OPENROUTER_MODEL")
	if openrouterModel == "" {
		openrouterModel = "meta-llama/llama-3.3-70b-instruct:free"
	}
	handlers.InitChat(openrouterKey, openrouterModel)

	spotify.Init(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"), os.Getenv("SPOTIFY_REFRESH_TOKEN"))

	app := fiber.New(fiber.Config{
		BodyLimit:    16 * 1024 * 1024,
		ErrorHandler: middleware.ErrorHandler,
	})

	app.Use(middleware.RequestContext())
	app.Use(middleware.AccessLog())
	app.Use(fiberrecover.New())
	app.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))
	app.Use(middleware.LangDetect())
	app.Static("/static", "./static", fiber.Static{
		Compress:      true,
		ByteRange:     true,
		MaxAge:        60 * 60 * 24 * 30, // 30 days; bust via filename if needed
		CacheDuration: 10 * 60,
	})

	// public
	app.Get("/", handlers.Home)
	app.Get("/og-image", handlers.OGImage)
	app.Post("/gift/claim", handlers.ClaimGift)
	app.Post("/chat", handlers.ChatStream)
	app.Get("/soundtrack/search", handlers.SoundtrackSearch)
	app.Post("/soundtrack/add", handlers.SoundtrackAdd)

	// admin group (must be registered before the /:code catch-all)
	admin := app.Group("/dashboard", middleware.BasicAuth())

	// csrf for all dashboard routes — token extracted from form field "_csrf"
	admin.Use(csrf.New(csrf.Config{
		Extractor: func(c *fiber.Ctx) (string, error) {
			if token := c.Get("X-Csrf-Token"); token != "" {
				return token, nil
			}
			return c.FormValue("_csrf"), nil
		},
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
	admin.Get("/registry/:id/edit", handlers.EditRegistryItemPage)
	admin.Post("/registry/:id", handlers.UpdateRegistryItem)
	admin.Post("/registry/:id/move/:direction", handlers.MoveRegistryItem)
	admin.Post("/registry/:id/delete", handlers.DeleteRegistryItem)
	admin.Get("/gifts/:id/edit", handlers.EditGiftPage)
	admin.Post("/gifts/:id", handlers.UpdateGift)
	admin.Post("/gifts/:id/delete", handlers.DeleteGift)
	admin.Post("/invitations", handlers.CreateInvitation)
	admin.Get("/invitations/:id/edit", handlers.EditInvitationPage)
	admin.Post("/invitations/:id", handlers.UpdateInvitation)
	admin.Post("/invitations/:id/delete", handlers.DeleteInvitation)
	admin.Post("/polls", handlers.AddPoll)
	admin.Get("/polls/:id/edit", handlers.EditPollPage)
	admin.Post("/polls/:id", handlers.UpdatePoll)
	admin.Post("/polls/:id/delete", handlers.DeletePoll)

	// invitation public routes (catch-all, must be last)
	app.Get("/:code", handlers.ViewInvitation)
	app.Post("/:code/viewed", handlers.MarkInvitationViewed)
	app.Post("/:code/rsvp", handlers.UpdateInvitationRSVP)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	addr := ":" + port
	slog.Info("starting server", "addr", addr)
	if err := app.Listen(addr); err != nil {
		slog.Error("server stopped", "error", err.Error())
		os.Exit(1)
	}
}
