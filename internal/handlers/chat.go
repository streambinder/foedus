package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/streambinder/foedus/internal/database"
)

const (
	chatRateLimit          = 10 // requests per minute per IP
	chatMaxMessageLen      = 500
	chatMaxHistoryInPrompt = 20
	chatMaxHistoryInReq    = 20
	chatMaxReplyTokens     = 300
)

var (
	openrouterKey   string
	openrouterModel string
	chatRateLimiter sync.Map // ip -> []time.Time
)

func InitChat(key, model string) {
	openrouterKey = key
	openrouterModel = model
	if key != "" {
		log.Printf("chat: enabled, model=%s", model)
	} else {
		log.Println("chat: disabled (OPENROUTER_API_KEY not set)")
	}
	go cleanRateLimiter()
}

func ChatEnabled() bool {
	return openrouterKey != ""
}

func cleanRateLimiter() {
	for range time.Tick(5 * time.Minute) {
		cutoff := time.Now().Add(-1 * time.Minute)
		chatRateLimiter.Range(func(k, v any) bool {
			times := v.([]time.Time)
			var recent []time.Time
			for _, t := range times {
				if t.After(cutoff) {
					recent = append(recent, t)
				}
			}
			if len(recent) == 0 {
				chatRateLimiter.Delete(k)
			} else {
				chatRateLimiter.Store(k, recent)
			}
			return true
		})
	}
}

func checkRateLimit(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-1 * time.Minute)
	raw, _ := chatRateLimiter.LoadOrStore(ip, []time.Time{})
	times := raw.([]time.Time)
	var recent []time.Time
	for _, t := range times {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= chatRateLimit {
		chatRateLimiter.Store(ip, recent)
		return false
	}
	chatRateLimiter.Store(ip, append(recent, now))
	return true
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Message string        `json:"message"`
	History []chatMessage `json:"history"`
}

func ChatStream(c *fiber.Ctx) error {
	if openrouterKey == "" {
		return c.SendStatus(fiber.StatusNotFound)
	}

	ip := c.IP()
	if !checkRateLimit(ip) {
		log.Printf("chat: rate limited ip=%s", ip)
		return c.Status(fiber.StatusTooManyRequests).SendString("rate limited")
	}

	var req chatRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		log.Printf("chat: bad json ip=%s err=%v", ip, err)
		return c.Status(fiber.StatusBadRequest).SendString("invalid json")
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" || len(req.Message) > chatMaxMessageLen {
		log.Printf("chat: invalid message len=%d ip=%s", len(req.Message), ip)
		return c.Status(fiber.StatusBadRequest).SendString("invalid message")
	}
	if len(req.History) > chatMaxHistoryInReq {
		log.Printf("chat: history too long len=%d ip=%s", len(req.History), ip)
		return c.Status(fiber.StatusBadRequest).SendString("history too long")
	}

	settings, err := database.GetAllSettings()
	if err != nil {
		log.Printf("chat: failed to load settings: %v", err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	if len(settings.Impersonations) == 0 {
		log.Printf("chat: no impersonations configured")
		return c.SendStatus(fiber.StatusNotFound)
	}

	persona := settings.Impersonations[rand.IntN(len(settings.Impersonations))]
	personaName := capitalizedPersonaName(persona.Codename)
	log.Printf("chat: request ip=%s persona=%q (pool=%d) msgLen=%d historyLen=%d", ip, persona.Codename, len(settings.Impersonations), len(req.Message), len(req.History))

	// build wedding context for system prompt
	playlistList := "none"
	if len(settings.SpotifyPlaylists) > 0 {
		playlistList = strings.Join(settings.SpotifyPlaylists, ", ")
	}
	var placeParts []string
	for _, p := range settings.Places {
		placeParts = append(placeParts, p.Label+": "+p.Name)
	}
	placeList := "none"
	if len(placeParts) > 0 {
		placeList = strings.Join(placeParts, ", ")
	}
	var accommodationParts []string
	for _, suggestion := range settings.AccommodationSuggestions {
		accommodationParts = append(accommodationParts, suggestion.Name)
	}
	accommodationList := "none"
	if len(accommodationParts) > 0 {
		accommodationList = strings.Join(accommodationParts, ", ")
	}

	lang := i18nLangFromAccept(c.Get("Accept-Language"))
	systemPrompt := fmt.Sprintf(
		"You are %s. You MUST write and behave exactly as described below:\n%s\n\n"+
			"You MUST sign every message with your name \"— %s\" at the very end.\n\n"+
			"Here is context about the wedding you can use to answer questions:\n"+
			"- Couple: %s & %s\n"+
			"- Ceremony: %s at %s, %s\n"+
			"- Reception: %s, %s\n"+
			"- Bank account (IBAN): %s, holder: %s\n"+
			"- Spotify playlists: %s\n"+
			"- Places of our story: %s\n\n"+
			"- Accommodation suggestions: %s\n\n"+
			"MEMORY RULE: The conversation history below is your memory. You must read it before replying. Do not ask the user to repeat information that is already present in earlier messages.\n\n"+
			"IDENTITY RULE: Never assume who you are talking to, but if the user already identified themselves anywhere in the conversation history, treat that identity as known and do not ask again. Only ask who they are when the conversation requires their identity and it is genuinely missing or ambiguous in the chat history.\n\n"+
			"SCOPE RULE: You only answer questions related to the wedding (date, location, logistics, couple, gifts, music, story, etc.). If someone asks about anything unrelated, politely decline and gently redirect the conversation back to the wedding.\n\n"+
			"Reply in the same language the user writes in. Preferred language hint: %s.\n"+
			"Keep replies warm, personal, and concise.",
		personaName, persona.Profile, personaName,
		settings.Spouse1Name, settings.Spouse2Name,
		settings.CeremonyDatetime, settings.CeremonyLocation, settings.CeremonyAddress,
		settings.ReceptionLocation, settings.ReceptionAddress,
		settings.BankAccountIBAN, settings.BankAccountHolder,
		playlistList, placeList, accommodationList, lang,
	)

	history := req.History
	if len(history) > chatMaxHistoryInPrompt {
		history = history[len(history)-chatMaxHistoryInPrompt:]
	}

	messages := []map[string]string{{"role": "system", "content": systemPrompt}}
	for _, m := range history {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		messages = append(messages, map[string]string{"role": m.Role, "content": m.Content})
	}
	messages = append(messages, map[string]string{"role": "user", "content": req.Message})

	payload, _ := json.Marshal(map[string]any{
		"model":      openrouterModel,
		"messages":   messages,
		"stream":     true,
		"max_tokens": chatMaxReplyTokens,
	})

	// set SSE headers before starting the stream writer — this lets the client
	// know immediately that data is coming while we wait for the upstream response
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // disable nginx buffering if behind a proxy

	start := time.Now()
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		upstreamReq, err := http.NewRequestWithContext(context.Background(), "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(payload))
		if err != nil {
			log.Printf("chat: failed to build upstream request: %v", err)
			fmt.Fprintf(w, "data: {\"error\":\"upstream error\"}\n\n")
			w.Flush()
			return
		}
		upstreamReq.Header.Set("Authorization", "Bearer "+openrouterKey)
		upstreamReq.Header.Set("Content-Type", "application/json")
		upstreamReq.Header.Set("HTTP-Referer", "https://foedus.wedding")

		resp, err := http.DefaultClient.Do(upstreamReq)
		if err != nil {
			log.Printf("chat: upstream request failed: %v", err)
			fmt.Fprintf(w, "data: {\"error\":\"upstream error\"}\n\n")
			w.Flush()
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("chat: upstream returned status=%d", resp.StatusCode)
			fmt.Fprintf(w, "data: {\"error\":\"upstream error\"}\n\n")
			w.Flush()
			return
		}

		log.Printf("chat: streaming started persona=%q ttfb=%s", persona.Codename, time.Since(start).Round(time.Millisecond))

		var chunks int
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			fmt.Fprintf(w, "%s\n\n", line)
			w.Flush()
			chunks++
			if strings.TrimSpace(strings.TrimPrefix(line, "data:")) == "[DONE]" {
				break
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("chat: stream scan error: %v", err)
			fmt.Fprintf(w, "data: {\"error\":\"stream error\"}\n\n")
			w.Flush()
		}
		log.Printf("chat: stream done persona=%q chunks=%d total=%s", persona.Codename, chunks, time.Since(start).Round(time.Millisecond))
	})

	return nil
}

// i18nLangFromAccept extracts a 2-char language code from Accept-Language header.
func i18nLangFromAccept(header string) string {
	for part := range strings.SplitSeq(header, ",") {
		tag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if len(tag) >= 2 {
			return strings.ToLower(tag[:2])
		}
	}
	return "en"
}

func capitalizedPersonaName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	r, size := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError && size == 0 {
		return ""
	}

	return string(unicode.ToUpper(r)) + name[size:]
}
