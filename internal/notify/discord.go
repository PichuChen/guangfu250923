package notify

import (
    "bytes"
    "context"
    "encoding/json"
    "log"
    "net/http"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

// SendDiscordWebhook sends a simple message to a Discord webhook URL.
// It expects the webhook URL as provided by Discord and a plain message string.
func SendDiscordWebhook(ctx context.Context, webhookURL, content string) error {
    if webhookURL == "" {
        return nil
    }
    payload := map[string]string{"content": content}
    b, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(b))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")
    // small timeout for webhook call
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        log.Printf("discord webhook returned status %d for url %s", resp.StatusCode, webhookURL)
    }
    return nil
}

// SendDiscordWebhookAsync fires-and-forgets the webhook call in a goroutine.
// Errors are logged but not returned to callers.
func SendDiscordWebhookAsync(webhookURL, content string) {
    if webhookURL == "" {
        return
    }
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
        defer cancel()
        if err := SendDiscordWebhook(ctx, webhookURL, content); err != nil {
            log.Printf("discord webhook error: %v", err)
        }
    }()
}

// SendDiscordWebhookAndRecordAsync sends the webhook and records the delivery result into
// webhook_deliveries table if pool != nil. resourceID and eventType are optional metadata.
func SendDiscordWebhookAndRecordAsync(pool *pgxpool.Pool, webhookURL, eventType, resourceID, content string, payload any) {
    if webhookURL == "" {
        return
    }
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
        defer cancel()

        // send
        var respStatus int
        var respBody string
        var sendErr error
        reqBody, _ := json.Marshal(map[string]string{"content": content})
        req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(reqBody))
        if err != nil {
            sendErr = err
        } else {
            req.Header.Set("Content-Type", "application/json")
            client := &http.Client{Timeout: 5 * time.Second}
            resp, err := client.Do(req)
            if err != nil {
                sendErr = err
            } else {
                respStatus = resp.StatusCode
                var b bytes.Buffer
                _, _ = b.ReadFrom(resp.Body)
                respBody = b.String()
                resp.Body.Close()
                if resp.StatusCode >= 300 {
                    log.Printf("discord webhook returned status %d for url %s", resp.StatusCode, webhookURL)
                }
            }
        }

        if pool == nil {
            if sendErr != nil {
                log.Printf("discord webhook error: %v", sendErr)
            }
            return
        }

        // record into DB (best-effort)
        payloadJSON, _ := json.Marshal(payload)
        // Use SQL with explicit parameter placeholders
        sql := `insert into webhook_deliveries (webhook_url,event_type,payload,response_status,response_body,error,resource_id) values ($1,$2,$3,$4,$5,$6,$7)`
        var err2 error
        if sendErr != nil {
            err2 = record(pool, sql, webhookURL, eventType, payloadJSON, respStatus, respBody, sendErr.Error(), resourceID)
        } else {
            err2 = record(pool, sql, webhookURL, eventType, payloadJSON, respStatus, respBody, sqlNullString(""), resourceID)
        }
        if err2 != nil {
            log.Printf("failed to record webhook_delivery: %v", err2)
        }
    }()
}

func record(pool *pgxpool.Pool, sqlStr string, webhookURL, eventType string, payloadJSON []byte, respStatus int, respBody string, errVal any, resourceID string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    // pgxpool doesn't accept []byte for jsonb directly in Exec; use string for simplicity
    _, err := pool.Exec(ctx, sqlStr, webhookURL, eventType, string(payloadJSON), respStatus, respBody, errVal, resourceID)
    return err
}

// helper to pass empty string or sql.NullString — provide simple wrapper returning empty string
func sqlNullString(s string) string { return s }
