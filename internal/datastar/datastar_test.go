package datastar

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func doRequest(t *testing.T, app *fiber.App, method, target, body string) (int, string, string) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	resp, err := app.Test(httptest.NewRequest(method, target, reader))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header.Get("Content-Type"), string(raw)
}

func TestPatchElements(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return PatchElements(c, "<div id=\"hal\">\n  Open\n</div>")
	})

	code, contentType, body := doRequest(t, app, "GET", "/", "")
	if code != 200 {
		t.Fatalf("status = %d", code)
	}
	if contentType != "text/event-stream" {
		t.Fatalf("Content-Type = %q", contentType)
	}
	want := "event: datastar-patch-elements\n" +
		"data: elements <div id=\"hal\">\n" +
		"data: elements   Open\n" +
		"data: elements </div>\n\n"
	if body != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}

func TestPatchElementsAt(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return PatchElementsAt(c, "#rows", "<tr><td>Ada</td></tr>")
	})

	_, _, body := doRequest(t, app, "GET", "/", "")
	want := "event: datastar-patch-elements\n" +
		"data: selector #rows\n" +
		"data: elements <tr><td>Ada</td></tr>\n\n"
	if body != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}

func TestPatchSignals(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return PatchSignals(c, map[string]any{"count": 2, "q": "ada"})
	})

	_, _, body := doRequest(t, app, "GET", "/", "")
	want := "event: datastar-patch-signals\ndata: signals {\"count\":2,\"q\":\"ada\"}\n\n"
	if body != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}

func TestPatchSignalsUnmarshalable(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return PatchSignals(c, func() {})
	})

	code, _, _ := doRequest(t, app, "GET", "/", "")
	if code != 500 {
		t.Fatalf("status = %d, want 500", code)
	}
}

type searchSignals struct {
	Q    string `json:"q"`
	Page int    `json:"page"`
}

func TestReadSignalsFromQuery(t *testing.T) {
	app := fiber.New()
	var got searchSignals
	app.Get("/", func(c *fiber.Ctx) error {
		if err := ReadSignals(c, &got); err != nil {
			return err
		}
		return c.SendString("ok")
	})

	code, _, _ := doRequest(t, app, "GET", "/?datastar=%7B%22q%22%3A%22ada%22%2C%22page%22%3A2%7D", "")
	if code != 200 {
		t.Fatalf("status = %d", code)
	}
	if got.Q != "ada" || got.Page != 2 {
		t.Fatalf("signals = %+v", got)
	}
}

func TestReadSignalsFromBody(t *testing.T) {
	app := fiber.New()
	var got searchSignals
	app.Post("/", func(c *fiber.Ctx) error {
		if err := ReadSignals(c, &got); err != nil {
			return err
		}
		return c.SendString("ok")
	})

	code, _, _ := doRequest(t, app, "POST", "/", `{"q":"bob","page":1}`)
	if code != 200 {
		t.Fatalf("status = %d", code)
	}
	if got.Q != "bob" || got.Page != 1 {
		t.Fatalf("signals = %+v", got)
	}
}

func TestReadSignalsAbsent(t *testing.T) {
	app := fiber.New()
	got := searchSignals{Q: "preset"}
	app.Get("/", func(c *fiber.Ctx) error {
		if err := ReadSignals(c, &got); err != nil {
			return err
		}
		return c.SendString("ok")
	})

	code, _, _ := doRequest(t, app, "GET", "/", "")
	if code != 200 {
		t.Fatalf("status = %d", code)
	}
	if got.Q != "preset" {
		t.Fatalf("signals overwritten: %+v", got)
	}
}

type stubComponent struct {
	html string
	err  error
}

func (s stubComponent) Render(_ context.Context, w io.Writer) error {
	if s.err != nil {
		return s.err
	}
	_, err := io.WriteString(w, s.html)
	return err
}

func TestPatchComponent(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return PatchComponent(c, stubComponent{html: "<div id=\"x\">hi</div>"})
	})

	_, contentType, body := doRequest(t, app, "GET", "/", "")
	if contentType != "text/event-stream" {
		t.Fatalf("Content-Type = %q", contentType)
	}
	want := "event: datastar-patch-elements\ndata: elements <div id=\"x\">hi</div>\n\n"
	if body != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}

func TestPatchComponentEvent(t *testing.T) {
	app := fiber.New()
	var got string
	var gotErr error
	app.Get("/", func(c *fiber.Ctx) error {
		got, gotErr = PatchComponentEvent(c, stubComponent{html: "<div id=\"x\">hi</div>"})
		return c.SendStatus(204)
	})

	code, _, _ := doRequest(t, app, "GET", "/", "")
	if code != 204 || gotErr != nil {
		t.Fatalf("status = %d, err = %v", code, gotErr)
	}
	want := "event: datastar-patch-elements\ndata: elements <div id=\"x\">hi</div>\n\n"
	if got != want {
		t.Fatalf("event = %q, want %q", got, want)
	}
}

func TestPatchComponentEventRenderError(t *testing.T) {
	app := fiber.New()
	var gotErr error
	app.Get("/", func(c *fiber.Ctx) error {
		_, gotErr = PatchComponentEvent(c, stubComponent{err: io.ErrClosedPipe})
		return c.SendStatus(204)
	})

	code, _, _ := doRequest(t, app, "GET", "/", "")
	if code != 204 {
		t.Fatalf("status = %d", code)
	}
	if gotErr == nil {
		t.Fatal("expected a render error, got nil")
	}
}

func TestPatchComponentRenderError(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return PatchComponent(c, stubComponent{err: io.ErrClosedPipe})
	})

	code, _, _ := doRequest(t, app, "GET", "/", "")
	if code != 500 {
		t.Fatalf("status = %d, want 500", code)
	}
}

func TestSendEventsMultiple(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		signals, err := PatchSignalsEvent(map[string]any{"page": 2})
		if err != nil {
			return err
		}
		return SendEvents(c, PatchElementsEvent("", "<p id=\"a\">x</p>"), signals)
	})

	_, _, body := doRequest(t, app, "GET", "/", "")
	want := "event: datastar-patch-elements\ndata: elements <p id=\"a\">x</p>\n\n" +
		"event: datastar-patch-signals\ndata: signals {\"page\":2}\n\n"
	if body != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}

func TestReadSignalsInvalid(t *testing.T) {
	app := fiber.New()
	var got searchSignals
	app.Get("/", func(c *fiber.Ctx) error {
		if err := ReadSignals(c, &got); err != nil {
			return c.Status(400).SendString("bad signals")
		}
		return c.SendString("ok")
	})

	code, _, _ := doRequest(t, app, "GET", "/?datastar=notjson", "")
	if code != 400 {
		t.Fatalf("status = %d, want 400", code)
	}
}
