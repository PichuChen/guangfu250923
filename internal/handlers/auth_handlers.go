package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// contract:
// - GET /auth/line/start?state=<frontend_state>&redirect_uri=<backend_redirect>
//   returns 302 to LINE authorize URL with signed state (JWT-like compact JWS, HMAC-SHA256)
// - POST /auth/line/token { code, state } -> exchanges code for access_token at LINE, validates state signature, returns tokens

// simple JWT-like compact token: base64url(header).base64url(payload).base64url(hmac)
// header is fixed: {"alg":"HS256","typ":"JWT"}
var headerB64 = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

type lineStatePayload struct {
	FrontendState string `json:"fs"`
	Exp           int64  `json:"exp"`
}

func (h *Handler) signState(p lineStatePayload) (string, error) {
	if os.Getenv("LINE_JWT_STATE_SECRET") == "" {
		return "", errors.New("missing JWT secret")
	}
	b, _ := json.Marshal(p)
	pb64 := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, []byte(os.Getenv("LINE_JWT_STATE_SECRET")))
	mac.Write([]byte(headerB64 + "." + pb64))
	sig := mac.Sum(nil)
	sb64 := base64.RawURLEncoding.EncodeToString(sig)
	return headerB64 + "." + pb64 + "." + sb64, nil
}

func (h *Handler) verifyState(tok string) (*lineStatePayload, error) {
	if os.Getenv("LINE_JWT_STATE_SECRET") == "" {
		return nil, errors.New("missing JWT secret")
	}
	parts := bytes.Split([]byte(tok), []byte{'.'})
	if len(parts) != 3 {
		return nil, errors.New("bad token format")
	}
	// verify header matches
	if !bytes.Equal(parts[0], []byte(headerB64)) {
		return nil, errors.New("bad header")
	}
	mac := hmac.New(sha256.New, []byte(os.Getenv("LINE_JWT_STATE_SECRET")))
	mac.Write([]byte(parts[0]))
	mac.Write([]byte{'.'})
	mac.Write([]byte(parts[1]))
	sig := mac.Sum(nil)
	expSig, err := base64.RawURLEncoding.DecodeString(string(parts[2]))
	if err != nil {
		return nil, err
	}
	if !hmac.Equal(sig, expSig) {
		return nil, errors.New("bad signature")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(string(parts[1]))
	if err != nil {
		return nil, err
	}
	var p lineStatePayload
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		return nil, err
	}
	if p.Exp > 0 && time.Now().Unix() > p.Exp {
		return nil, errors.New("expired state")
	}
	return &p, nil
}

// StartLineAuth builds a signed state and redirects to LINE authorize endpoint.
func (h *Handler) StartLineAuth(c *gin.Context) {
	if os.Getenv("LINE_CHANNEL_ID") == "" || os.Getenv("LINE_REDIRECT_URI") == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LINE config missing"})
		return
	}
	frontendState := c.Query("state")
	exp := time.Now().Add(10 * time.Minute).Unix()
	tok, err := h.signState(lineStatePayload{FrontendState: frontendState, Exp: exp})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	feRedirectURI := c.Query("redirect_uri")
	// LINE authorize URL
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("client_id", os.Getenv("LINE_CHANNEL_ID"))
	if feRedirectURI != "" {
		v.Set("redirect_uri", feRedirectURI)
	} else {
		v.Set("redirect_uri", os.Getenv("LINE_REDIRECT_URI"))
	}
	v.Set("state", tok)
	v.Set("scope", "profile openid email")
	authURL := "https://access.line.me/oauth2/v2.1/authorize?" + v.Encode()
	c.Redirect(http.StatusFound, authURL)
}

type lineTokenReq struct {
	Code        string  `json:"code"`
	State       string  `json:"state"`
	RedirectURI *string `json:"redirect_uri"` // optional, if differs from env
}

type lineTokenResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

// ExchangeLineToken validates state and exchanges code for tokens via LINE API.
func (h *Handler) ExchangeLineToken(c *gin.Context) {
	if os.Getenv("LINE_CHANNEL_ID") == "" || os.Getenv("LINE_CHANNEL_SECRET") == "" || os.Getenv("LINE_REDIRECT_URI") == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LINE config missing"})
		return
	}
	var in lineTokenReq
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.Code == "" || in.State == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and state are required"})
		return
	}
	if _, err := h.verifyState(in.State); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state: " + err.Error()})
		return
	}
	// Exchange at LINE token endpoint
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", in.Code)
	if in.RedirectURI != nil && *in.RedirectURI != "" {
		form.Set("redirect_uri", *in.RedirectURI)
	} else {
		form.Set("redirect_uri", os.Getenv("LINE_REDIRECT_URI"))
	}
	form.Set("client_id", os.Getenv("LINE_CHANNEL_ID"))
	form.Set("client_secret", os.Getenv("LINE_CHANNEL_SECRET"))

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.line.me/oauth2/v2.1/token", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		c.JSON(http.StatusBadGateway, gin.H{"error": "line token error", "status": resp.StatusCode, "body": body})
		return
	}
	var out lineTokenResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	// Return raw tokens to frontend to continue its flow.
	c.JSON(http.StatusOK, out)
}
