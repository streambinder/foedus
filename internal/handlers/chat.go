package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
		slog.Info("chat service enabled", "model", model)
	} else {
		slog.Warn("chat service disabled", "reason", "OPENROUTER_API_KEY not set")
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
	logger := handlerLogger(c)

	if openrouterKey == "" {
		logger.Warn("chat request rejected", "reason", "service disabled")
		return c.SendStatus(fiber.StatusNotFound)
	}

	ip := c.IP()
	if !checkRateLimit(ip) {
		logger.Warn("chat request rate limited")
		return c.Status(fiber.StatusTooManyRequests).SendString("rate limited")
	}

	var req chatRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		logger.Warn("chat request invalid json", "error", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString("invalid json")
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" || len(req.Message) > chatMaxMessageLen {
		logger.Warn("chat request invalid message", "message_len", len(req.Message))
		return c.Status(fiber.StatusBadRequest).SendString("invalid message")
	}
	if len(req.History) > chatMaxHistoryInReq {
		logger.Warn("chat request history too long", "history_len", len(req.History))
		return c.Status(fiber.StatusBadRequest).SendString("history too long")
	}

	settings, err := database.GetAllSettings()
	if err != nil {
		logger.Error("chat request failed to load settings", "error", err.Error())
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	if len(settings.Impersonations) == 0 {
		logger.Warn("chat request rejected", "reason", "no impersonations configured")
		return c.SendStatus(fiber.StatusNotFound)
	}

	persona := settings.Impersonations[rand.IntN(len(settings.Impersonations))]
	personaName := capitalizedPersonaName(persona.Codename)
	logger.Info(
		"chat request accepted",
		"persona", persona.Codename,
		"persona_pool", len(settings.Impersonations),
		"message_len", len(req.Message),
		"history_len", len(req.History),
	)

	// build wedding context for system prompt
	playlistList := "none"
	if settings.SpotifyPlaylist != "" {
		playlistList = settings.SpotifyPlaylist
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

	history := req.History
	if len(history) > chatMaxHistoryInPrompt {
		history = history[len(history)-chatMaxHistoryInPrompt:]
	}
	systemPrompt := fmt.Sprintf(
		"You are %s.\n\n"+
			"PERSONA STYLE RULE: The profile below is tone reference only. It defines persona, rhythm, and attitude, but it is not a language instruction. Apply its style in the language chosen from the user's latest message; never copy the profile's language just because it appears there.\n"+
			"STYLE TRANSFER RULE: Extract abstract style traits from the profile, such as energy, warmth, brevity, humor, pacing, and directness, then re-express those traits natively in the inbound message language. Treat the profile as foreign-language source material for style only: keep its meaning, discard its wording. Do not import slang, filler words, catchphrases, or sentence fragments from the profile into another language unless the user used them first.\n"+
			"<persona_profile>\n%s\n</persona_profile>\n\n"+
			"You MUST sign every message with your name \"— %s\" at the very end.\n\n"+
			"Here is context about the wedding you can use to answer questions:\n"+
			"- Couple: %s & %s\n"+
			"- Ceremony time: %s\n"+
			"- Ceremony venue: %s, %s\n"+
			"- Reception time: %s\n"+
			"- Reception venue: %s, %s\n"+
			"- Bank account (IBAN): %s, holder: %s\n"+
			"- Spotify playlists: %s\n"+
			"- Places of our story: %s\n\n"+
			"- Accommodation suggestions: %s\n\n"+
			"MEMORY RULE: The conversation history below is your memory. You must read it before replying. Do not ask the user to repeat information that is already present in earlier messages.\n\n"+
			"IDENTITY RULE: Never assume who you are talking to, but if the user already identified themselves anywhere in the conversation history, treat that identity as known and do not ask again. Only ask who they are when the conversation requires their identity and it is genuinely missing or ambiguous in the chat history.\n\n"+
			"SCOPE RULE: You only answer questions related to the wedding (date, location, logistics, couple, gifts, music, story, etc.). If someone asks about anything unrelated, politely decline and gently redirect the conversation back to the wedding.\n\n"+
			"LANGUAGE RULE: Always reply in the same language as the most recent user message. Use only the current inbound message to determine reply language. Never use past user messages, past assistant messages, or any other context to infer reply language.\n"+
			"CONTEXT RULE: The conversation context below contains past messages that have already been handled. Use it only as memory and factual reference. Do not reply to those messages again, and do not use them for language inference.\n"+
			"INBOUND RULE: There is exactly one current inbound user message to answer now. Reply only to that message.\n\n"+
			"PAST CONTEXT ALREADY HANDLED:\n<conversation_context>\n%s\n</conversation_context>\n\n"+
			"CURRENT INBOUND MESSAGE TO ANSWER:\n<inbound_message>\n%s\n</inbound_message>\n\n"+
			"FINAL INSTRUCTION: Read only the inbound message as the message that needs a reply. Use the past context only for reference.\n"+
			"Follow this procedure silently before answering:\n"+
			"1. Identify the language of the inbound message from its text.\n"+
			"2. If the inbound message language is ambiguous from the inbound message alone, ask a very short clarification instead of guessing from prior context.\n"+
			"3. Otherwise, write the reply only in the inbound message language.\n"+
			"4. Audit the draft and rewrite it if it contains words from another language, except names, quoted user text, fixed place names, or the required signature.\n"+
			"5. Output only the final reply.\n"+
			"Do not code-switch. Do not mix languages. Do not use prior messages to decide the reply language. The visible reply must be entirely in the inbound message language except for the allowed exceptions above. End the reply with the exact signature \"— %s\".\n"+
			"Keep replies warm, personal, and concise.",
		personaName, persona.Profile, personaName,
		settings.Spouse1Name, settings.Spouse2Name,
		settings.CeremonyDatetime, settings.CeremonyLocation, settings.CeremonyAddress,
		settings.ReceptionDatetime, settings.ReceptionLocation, settings.ReceptionAddress,
		settings.BankAccountIBAN, settings.BankAccountHolder,
		playlistList, placeList, accommodationList, formatConversationContext(history), req.Message, personaName,
	)

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": req.Message},
	}

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
		streamLogger := logger.With("persona", persona.Codename)

		upstreamReq, err := http.NewRequestWithContext(context.Background(), "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(payload))
		if err != nil {
			streamLogger.Error("chat upstream request build failed", "error", err.Error())
			fmt.Fprintf(w, "data: {\"error\":\"upstream error\"}\n\n")
			w.Flush()
			return
		}
		upstreamReq.Header.Set("Authorization", "Bearer "+openrouterKey)
		upstreamReq.Header.Set("Content-Type", "application/json")
		upstreamReq.Header.Set("HTTP-Referer", "https://foedus.wedding")

		resp, err := http.DefaultClient.Do(upstreamReq)
		if err != nil {
			streamLogger.Error("chat upstream request failed", "error", err.Error())
			fmt.Fprintf(w, "data: {\"error\":\"upstream error\"}\n\n")
			w.Flush()
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			streamLogger.Error("chat upstream returned unexpected status", "status", resp.StatusCode)
			fmt.Fprintf(w, "data: {\"error\":\"upstream error\"}\n\n")
			w.Flush()
			return
		}

		streamLogger.Info("chat streaming started", "ttfb_ms", time.Since(start).Milliseconds())

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
			streamLogger.Error("chat stream scan failed", "chunks", chunks, "error", err.Error())
			fmt.Fprintf(w, "data: {\"error\":\"stream error\"}\n\n")
			w.Flush()
		}
		streamLogger.Info("chat stream completed", "chunks", chunks, "duration_ms", time.Since(start).Milliseconds())
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

func formatConversationContext(history []chatMessage) string {
	if len(history) == 0 {
		return "(none)"
	}

	var lines []string
	for idx, m := range history {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}

		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}

		lines = append(lines, fmt.Sprintf("%d. [%s][already handled] %s", idx+1, m.Role, content))
	}

	if len(lines) == 0 {
		return "(none)"
	}

	return strings.Join(lines, "\n")
}
