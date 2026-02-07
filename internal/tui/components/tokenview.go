package components

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jwtea/internal/tui/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/golang-jwt/jwt/v5"
)

type TokenView struct {
	Token       string
	ShowDecoded bool
	ExpiresAt   time.Time
	Width       int
	Focused     bool

	styleHeader  lipgloss.Style
	styleToken   lipgloss.Style
	styleExpiry  lipgloss.Style
	styleFocused lipgloss.Style
}

func NewTokenView(token string) *TokenView {
	tv := &TokenView{
		Token:        token,
		ShowDecoded:  false,
		styleHeader:  theme.Header,
		styleToken:   theme.Text,
		styleExpiry:  theme.Muted,
		styleFocused: theme.Accent,
	}

	if token != "" {
		if claims, err := parseTokenClaims(token); err == nil {
			if exp, ok := claims["exp"].(float64); ok {
				tv.ExpiresAt = time.Unix(int64(exp), 0)
			}
		}
	}

	return tv
}

func (t *TokenView) Init() tea.Cmd {
	return nil
}

func (t *TokenView) Update(msg tea.Msg) (*TokenView, tea.Cmd) {
	return t, nil
}

func (t *TokenView) View() string {
	if t.Token == "" {
		return t.styleHeader.Render("No token generated yet")
	}

	var b strings.Builder

	headerStyle := t.styleHeader
	if t.Focused {
		headerStyle = t.styleFocused.Bold(true)
	}

	focusIndicator := "  "
	if t.Focused {
		focusIndicator = "▶ "
	}

	b.WriteString(focusIndicator + headerStyle.Render("Generated Token:"))
	b.WriteString("\n\n")

	wrapWidth := t.Width - 4
	if wrapWidth < 40 {
		wrapWidth = 40
	}
	if wrapWidth > 120 {
		wrapWidth = 120
	}
	wrapped := wordWrap(t.Token, wrapWidth)
	b.WriteString(t.styleToken.Render(wrapped))
	b.WriteString("\n\n")

	if t.ShowDecoded {
		decoded := t.decodeToken()
		b.WriteString(focusIndicator + headerStyle.Render("Decoded:"))
		b.WriteString("\n")
		b.WriteString(decoded)
		b.WriteString("\n\n")
		if t.Focused {
			b.WriteString(t.styleFocused.Render("d toggle decoded • c copy token"))
		} else {
			b.WriteString(t.styleExpiry.Render("Press 'd' to hide decoded view"))
		}
	} else {
		if !t.ExpiresAt.IsZero() {
			remaining := time.Until(t.ExpiresAt)
			expiryText := ""
			if remaining > 0 {
				expiryText = fmt.Sprintf("Expires in: %s", formatDuration(remaining))
			} else {
				expiryText = fmt.Sprintf("Expired %s ago", formatDuration(-remaining))
			}
			b.WriteString(t.styleExpiry.Render(expiryText))
			b.WriteString("\n\n")
		}
		if t.Focused {
			b.WriteString(t.styleFocused.Render("d toggle decoded • c copy token"))
		} else {
			b.WriteString(t.styleExpiry.Render("Press 'd' to toggle decoded view"))
		}
	}

	return b.String()
}

func (t *TokenView) ToggleDecoded() {
	t.ShowDecoded = !t.ShowDecoded
}

func (t *TokenView) SetToken(token string) {
	t.Token = token
	t.ShowDecoded = false

	if token != "" {
		if claims, err := parseTokenClaims(token); err == nil {
			if exp, ok := claims["exp"].(float64); ok {
				t.ExpiresAt = time.Unix(int64(exp), 0)
			}
		}
	}
}

func (t *TokenView) SetWidth(w int) {
	t.Width = w
}

func (t *TokenView) SetFocused(focused bool) {
	t.Focused = focused
}

func (t *TokenView) decodeToken() string {
	parts := strings.Split(t.Token, ".")
	if len(parts) != 3 {
		return "Invalid JWT format"
	}

	var b strings.Builder

	if header, err := decodeJWTPart(parts[0]); err == nil {
		b.WriteString("Header:\n")
		b.WriteString(formatJSON(header))
		b.WriteString("\n\n")
	}

	if payload, err := decodeJWTPart(parts[1]); err == nil {
		b.WriteString("Payload:\n")
		b.WriteString(formatJSON(payload))
	}

	return b.String()
}

func parseTokenClaims(tokenString string) (jwt.MapClaims, error) {
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid claims type")
}

func decodeJWTPart(part string) (map[string]any, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(decoded, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func formatJSON(data map[string]any) string {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return string(bytes)
}

func wordWrap(text string, width int) string {
	if len(text) <= width {
		return text
	}

	var b strings.Builder
	for i := 0; i < len(text); i += width {
		end := i + width
		if end > len(text) {
			end = len(text)
		}
		b.WriteString(text[i:end])
		if end < len(text) {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d / time.Minute)
		s := int((d % time.Minute) / time.Second)
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := int(d / time.Hour)
	m := int((d % time.Hour) / time.Minute)
	return fmt.Sprintf("%dh%dm", h, m)
}
