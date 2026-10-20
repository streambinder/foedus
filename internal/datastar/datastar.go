// Package datastar writes the subset of the Datastar server protocol this
// app uses: SSE patch events and signal decoding, on top of Fiber.
//
// The official Go SDK targets net/http; foedus serves through Fiber/fasthttp,
// where the SDK cannot write responses without an adaptor detour. The wire
// format is a documented, stable protocol (see data-star.dev/guide), so the
// few events needed here are written directly and pinned by unit tests.
package datastar

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// PatchElements sends a datastar-patch-elements SSE event. The elements are
// morphed into the DOM by element ID (Datastar's default outer morph).
func PatchElements(c *fiber.Ctx, elements string) error {
	return send(c, patchElementsEvent("", elements))
}

// PatchElementsAt sends a datastar-patch-elements SSE event that morphs the
// elements into the element matching the given CSS selector.
func PatchElementsAt(c *fiber.Ctx, selector, elements string) error {
	return send(c, patchElementsEvent(selector, elements))
}

func patchElementsEvent(selector, elements string) string {
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

// PatchSignals sends a datastar-patch-signals SSE event carrying the given
// value encoded as JSON.
func PatchSignals(c *fiber.Ctx, signals any) error {
	raw, err := json.Marshal(signals)
	if err != nil {
		return err
	}
	return send(c, "event: datastar-patch-signals\ndata: signals "+string(raw)+"\n\n")
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

func send(c *fiber.Ctx, event string) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	return c.SendString(event)
}
