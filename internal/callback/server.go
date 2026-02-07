package callback

import (
	"fmt"
	"net/http"

	"jwtea/internal/config"
)

type Handler struct {
	config      config.CallbackServer
	oauthServer string
}

func NewHandler(cfg config.CallbackServer, oauthServer string) *Handler {
	return &Handler{
		config:      cfg,
		oauthServer: oauthServer,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handleCallback(w, r)
}

func (h *Handler) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")
	errorDesc := r.URL.Query().Get("error_description")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if errorParam != "" {
		h.renderError(w, errorParam, errorDesc)
		return
	}

	if code == "" {
		h.renderError(w, "invalid_request", "Missing authorization code")
		return
	}

	h.renderSuccess(w, code, state)
}

func (h *Handler) renderSuccess(w http.ResponseWriter, code, state string) {
	stateDisplay := state
	if stateDisplay == "" {
		stateDisplay = "(none)"
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OAuth Callback</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            max-width: 800px;
            margin: 40px auto;
            padding: 20px;
            line-height: 1.6;
        }
        h1 { color: #2d3748; margin-bottom: 20px; font-size: 24px; }
        .success { color: #38a169; font-weight: 600; }
        .section {
            background: #f7fafc;
            border: 1px solid #e2e8f0;
            border-radius: 6px;
            padding: 16px;
            margin-bottom: 16px;
        }
        .label {
            font-size: 12px;
            font-weight: 600;
            color: #718096;
            text-transform: uppercase;
            margin-bottom: 8px;
        }
        .value {
            font-family: 'Courier New', monospace;
            font-size: 14px;
            color: #2d3748;
            word-break: break-all;
            background: white;
            padding: 12px;
            border-radius: 4px;
            border: 1px solid #e2e8f0;
        }
        button {
            background: #4299e1;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 4px;
            font-size: 13px;
            cursor: pointer;
            margin-top: 8px;
        }
        button:hover { background: #3182ce; }
        .footer {
            margin-top: 32px;
            padding-top: 16px;
            border-top: 1px solid #e2e8f0;
            font-size: 13px;
            color: #718096;
        }
        .footer a {
            color: #4299e1;
            text-decoration: none;
        }
    </style>
</head>
<body>
    <h1><span class="success">✓</span> Authorization Successful</h1>

    <div class="section">
        <div class="label">Authorization Code</div>
        <div class="value" id="code">%s</div>
        <button onclick="copy('code')">Copy Code</button>
    </div>

    <div class="section">
        <div class="label">State</div>
        <div class="value">%s</div>
    </div>

    <div class="footer">
        Use the code above with your OAuth client to exchange for an access token.<br>
        Token endpoint: <a href="%s/oauth2/token">%s/oauth2/token</a>
    </div>

    <script>
        function copy(id) {
            const el = document.getElementById(id);
            navigator.clipboard.writeText(el.textContent).then(() => {
                const btn = event.target;
                const orig = btn.textContent;
                btn.textContent = '✓ Copied';
                setTimeout(() => { btn.textContent = orig; }, 2000);
            });
        }
    </script>
</body>
</html>`, code, stateDisplay, h.oauthServer, h.oauthServer)

	_, err := fmt.Fprint(w, html)
	if err != nil {
		return
	}
}

func (h *Handler) renderError(w http.ResponseWriter, errorCode, errorDesc string) {
	if errorDesc == "" {
		errorDesc = "No description provided"
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OAuth Error</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            max-width: 800px;
            margin: 40px auto;
            padding: 20px;
            line-height: 1.6;
        }
        h1 { color: #2d3748; margin-bottom: 20px; font-size: 24px; }
        .error { color: #e53e3e; font-weight: 600; }
        .section {
            background: #fff5f5;
            border: 1px solid #feb2b2;
            border-radius: 6px;
            padding: 16px;
            margin-bottom: 16px;
        }
        .code {
            font-family: 'Courier New', monospace;
            font-size: 14px;
            font-weight: 600;
            color: #c53030;
            margin-bottom: 8px;
        }
        .desc {
            font-size: 14px;
            color: #742a2a;
        }
    </style>
</head>
<body>
    <h1><span class="error">✗</span> Authorization Failed</h1>

    <div class="section">
        <div class="code">%s</div>
        <div class="desc">%s</div>
    </div>
</body>
</html>`, errorCode, errorDesc)

	_, err := fmt.Fprint(w, html)
	if err != nil {
		return
	}
}
