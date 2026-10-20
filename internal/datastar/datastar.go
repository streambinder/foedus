// Package datastar writes the subset of the Datastar server protocol this
// app uses: SSE patch events and signal decoding, on top of Fiber.
//
// The official Go SDK targets net/http; foedus serves through Fiber/fasthttp,
// where the SDK cannot write responses without an adaptor detour. The wire
// format is a documented, stable protocol (see data-star.dev/guide), so the
// few events needed here are written directly and pinned by unit tests.
package datastar

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
)

// PatchElementsEvent builds a datastar-patch-elements SSE event. The
// elements are morphed into the DOM by element ID, or into the element
// matching selector when one is given.
func PatchElementsEvent(selector, elements string) string {
	var b strings.Builder
	b.WriteString("event: datastar-patch-elements\n")
	if selector != "" {
		b.WriteString("data: selector " + selector + "\n")
	}
	for _, line := range strings.Split(elements, "\n") {
		b.WriteString("data: elements " + line + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

// PatchSignalsEvent builds a datastar-patch-signals SSE event carrying the
// given value encoded as JSON.
func PatchSignalsEvent(signals any) (string, error) {
	raw, err := json.Marshal(signals)
	if err != nil {
		return "", err
	}
	return "event: datastar-patch-signals\ndata: signals " + string(raw) + "\n\n", nil
}

// SendEvents writes one or more SSE events as a single response.
func SendEvents(c *fiber.Ctx, events ...string) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	return c.SendString(strings.Join(events, ""))
}

// PatchElements sends a datastar-patch-elements SSE event. The elements are
// morphed into the DOM by element ID (Datastar's default outer morph).
func PatchElements(c *fiber.Ctx, elements string) error {
	return SendEvents(c, PatchElementsEvent("", elements))
}

// PatchElementsAt sends a datastar-patch-elements SSE event that morphs the
// elements into the element matching the given CSS selector.
func PatchElementsAt(c *fiber.Ctx, selector, elements string) error {
	return SendEvents(c, PatchElementsEvent(selector, elements))
}

// PatchComponentEvent renders a templ component into a
// datastar-patch-elements SSE event. Handlers that patch several
// components in one response build one event per component and send them
// together with SendEvents.
func PatchComponentEvent(c *fiber.Ctx, comp templ.Component) (string, error) {
	var buf bytes.Buffer
	if err := comp.Render(c.Context(), &buf); err != nil {
		return "", err
	}
	return PatchElementsEvent("", buf.String()), nil
}

// PatchComponent renders a templ component and sends it as a
// datastar-patch-elements SSE event.
func PatchComponent(c *fiber.Ctx, comp templ.Component) error {
	event, err := PatchComponentEvent(c, comp)
	if err != nil {
		return err
	}
	return SendEvents(c, event)
}

// PatchSignals sends a datastar-patch-signals SSE event carrying the given
// value encoded as JSON.
func PatchSignals(c *fiber.Ctx, signals any) error {
	event, err := PatchSignalsEvent(signals)
	if err != nil {
		return err
	}
	return SendEvents(c, event)
}

// ReadSignals decodes the signals a Datastar action sent into out. GET
// requests carry them in the "datastar" query parameter, other methods in
// the request body. A request without signals leaves out untouched.
func ReadSignals(c *fiber.Ctx, out any) error {
	raw := c.Query("datastar")
	if raw == "" && c.Method() != fiber.MethodGet {
		raw = string(c.Body())
	}
	if raw == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), out)
}
