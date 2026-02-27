package handlers

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/dpucci/foedus/internal/database"
	"github.com/dpucci/foedus/templates"
	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)

func StripeEnabled() bool {
	return os.Getenv("STRIPE_SECRET_KEY") != ""
}

func stripeCurrency() string {
	if c := os.Getenv("STRIPE_CURRENCY"); c != "" {
		return strings.ToLower(c)
	}
	return "eur"
}

// InitStripe sets the global stripe key once at startup
func InitStripe() {
	if key := os.Getenv("STRIPE_SECRET_KEY"); key != "" {
		stripe.Key = key
	}
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

func CreateGiftCheckout(c *fiber.Ctx) error {
	if !StripeEnabled() {
		return c.Status(404).SendString("payments not configured")
	}

	amountStr := strings.TrimSpace(c.FormValue("amount"))
	if amountStr == "" {
		return c.Status(400).SendString("amount is required")
	}

	amountFloat, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amountFloat < 1 || amountFloat > 10000 {
		return c.Status(400).SendString("invalid amount")
	}
	amountCents := int64(math.Round(amountFloat * 100))

	donor := truncate(strings.TrimSpace(c.FormValue("donor")), 200)
	message := truncate(strings.TrimSpace(c.FormValue("message")), 400)

	currency := stripeCurrency()
	baseURL := schemeHost(c)

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String(currency),
				UnitAmount: stripe.Int64(amountCents),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String("Wedding Gift"),
				},
			},
			Quantity: stripe.Int64(1),
		}},
		SuccessURL: stripe.String(baseURL + "/gift/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(baseURL + "/gift/cancel"),
		Metadata: map[string]string{
			"donor":   donor,
			"message": message,
		},
	}

	s, err := session.New(params)
	if err != nil {
		log.Printf("stripe checkout session creation failed: %v", err)
		return c.Status(500).SendString("failed to create checkout session")
	}

	return c.Redirect(s.URL, fiber.StatusSeeOther)
}

func GiftSuccess(c *fiber.Ctx) error {
	if !StripeEnabled() {
		return c.Status(404).SendString("payments not configured")
	}

	sessionID := c.Query("session_id")
	if sessionID == "" {
		return c.Redirect("/")
	}

	s, err := session.Get(sessionID, nil)
	if err != nil || s.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		return c.Redirect("/")
	}

	currency := string(s.Currency)
	donor := s.Metadata["donor"]
	message := s.Metadata["message"]
	if err := database.CreateGift(int(s.AmountTotal), currency, donor, message, sessionID); err != nil {
		log.Printf("failed to store gift (session %s): %v", sessionID, err)
	}

	settings, _ := database.GetAllSettings()
	return Render(c, templates.GiftSuccess(settings, donor, fmt.Sprintf("%.2f", float64(s.AmountTotal)/100), strings.ToUpper(currency), getT(c), getLang(c)))
}

func GiftCancel(c *fiber.Ctx) error {
	return c.Redirect("/")
}

func schemeHost(c *fiber.Ctx) string {
	return c.Protocol() + "://" + c.Hostname()
}
