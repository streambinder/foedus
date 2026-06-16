//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

package main

import (
	"log/slog"
	"os"
	"strings"
	"time"

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

	// trust the loopback reverse proxy (nginx on the same host) so c.IP() returns
	// the real client IP from X-Forwarded-For instead of 127.0.0.1. Without this,
	// every per-IP rate limiter collapses to a single bucket.
	trustedProxies := []string{"127.0.0.1", "::1"}
	if extra := os.Getenv("TRUSTED_PROXIES"); extra != "" {
		for p := range strings.SplitSeq(extra, ",") {
			if p = strings.TrimSpace(p); p != "" {
				trustedProxies = append(trustedProxies, p)
			}
		}
	}

	app := fiber.New(fiber.Config{
		// admin /dashboard/settings POSTs base64 image fields; keep generous
		// global cap, but restrict public POSTs via publicBodyLimit below.
		BodyLimit:               32 * 1024 * 1024,
		ReadTimeout:             15 * time.Second,
		WriteTimeout:            120 * time.Second, // long enough for SSE streaming reply
		IdleTimeout:             60 * time.Second,
		ErrorHandler:            middleware.ErrorHandler,
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          trustedProxies,
	})

	// rejects public POSTs with body > 256KB before fasthttp materializes them.
	// Content-Length is set by fasthttp from the parsed request line; missing
	// or negative means chunked/unknown — reject conservatively.
	publicBodyLimit := func(c *fiber.Ctx) error {
		const maxBytes = 256 * 1024
		if c.Method() == fiber.MethodPost {
			if cl := c.Request().Header.ContentLength(); cl < 0 || cl > maxBytes {
				return c.Status(fiber.StatusRequestEntityTooLarge).SendString("payload too large")
			}
		}
		return c.Next()
	}

	app.Use(middleware.RequestContext())
	app.Use(middleware.AccessLog())
	app.Use(fiberrecover.New())
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
		Next: func(c *fiber.Ctx) bool {
			// already-compressed binary: skip re-compression to save CPU
			return strings.HasPrefix(c.Path(), "/media/")
		},
	}))
	app.Use(middleware.LangDetect())
	app.Use(middleware.SecurityHeaders())
	app.Static("/static", "./static", fiber.Static{
		Compress:      true,
		ByteRange:     true,
		MaxAge:        60 * 60 * 24 * 30, // 30 days; bust via filename if needed
		CacheDuration: 10 * time.Minute,
	})
	app.Get("/robots.txt", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/plain")
		return c.SendString("User-agent: *\nAllow: /\n")
	})

	// public CSRF: cookie issued on every GET, validated on every state-changing
	// public POST. Token exposed via Locals("csrf") so templates inject it into
	// the RSVP <form>; JS endpoints read it from the cookie and send X-Csrf-Token.
	publicCSRF := csrf.New(csrf.Config{
		Extractor: func(c *fiber.Ctx) (string, error) {
			if token := c.Get("X-Csrf-Token"); token != "" {
				return token, nil
			}
			return c.FormValue("_csrf"), nil
		},
		ContextKey:     "csrf",
		CookieName:     "csrf_public",
		CookieSameSite: "Lax",
		CookieSecure:   true,
		CookieHTTPOnly: false, // JS reads it on /chat and /soundtrack
		Expiration:     12 * time.Hour,
	})

	// public
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Get("/", publicCSRF, handlers.Home)
	app.Get("/ceremony", handlers.Ceremony)
	app.Get("/media/:id", handlers.MediaImage)
	app.Get("/og-image", handlers.OGImage)
	app.Post("/gift/claim", publicBodyLimit, publicCSRF, handlers.ClaimGift)
	app.Post("/chat", publicBodyLimit, publicCSRF, handlers.ChatStream)
	app.Get("/soundtrack/search", handlers.SoundtrackSearch)
	app.Post("/soundtrack/add", publicBodyLimit, publicCSRF, handlers.SoundtrackAdd)

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
	admin.Get("/counters/:category", handlers.CounterGuestNames)
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
	admin.Post("/invitations/:id/viewed/reset", handlers.ResetInvitationViewed)
	admin.Post("/polls", handlers.AddPoll)
	admin.Get("/polls/:id/edit", handlers.EditPollPage)
	admin.Post("/polls/:id", handlers.UpdatePoll)
	admin.Post("/polls/:id/delete", handlers.DeletePoll)
	admin.Post("/soundtrack/:id/delete", handlers.DeleteSoundtrackEvent)

	// invitation public routes (catch-all, must be last)
	app.Get("/:code", publicCSRF, handlers.ViewInvitation)
	app.Post("/:code/viewed", publicBodyLimit, publicCSRF, handlers.MarkInvitationViewed)
	app.Post("/:code/rsvp", publicBodyLimit, publicCSRF, handlers.UpdateInvitationRSVP)

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
