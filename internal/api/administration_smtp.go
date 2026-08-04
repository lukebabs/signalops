package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lukebabs/signalops/internal/storage"
)

func registerAdministrationSMTPRoutes(mux *http.ServeMux, cfg RouterConfig) {
	repo, ok := cfg.QueryRepository.(storage.AdministrationNotificationRepository)
	if !ok {
		return
	}
	mux.HandleFunc("GET /v1/administration/notification-email", func(w http.ResponseWriter, r *http.Request) {
		if !requireNotificationSuperAdmin(w, r, cfg) {
			return
		}
		tenant := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		if tenant == "" {
			writeError(w, 400, "tenant_id_required", "tenant_id is required")
			return
		}
		settings, err := repo.GetAdministrationSMTPSettings(r.Context(), tenant)
		if err == storage.ErrNotFound {
			writeJSON(w, 200, map[string]any{"configured": false})
			return
		}
		if err != nil {
			writeError(w, 500, "query_failed", "failed to load email settings")
			return
		}
		writeJSON(w, 200, map[string]any{"configured": true, "settings": settings})
	})
	mux.HandleFunc("PUT /v1/administration/notification-email", func(w http.ResponseWriter, r *http.Request) {
		p, _ := principalFromContext(r.Context())
		if !requireNotificationSuperAdmin(w, r, cfg) {
			return
		}
		var req struct {
			TenantID    string `json:"tenant_id"`
			Host        string `json:"host"`
			Username    string `json:"username"`
			Password    string `json:"password"`
			FromEmail   string `json:"from_email"`
			FromName    string `json:"from_name"`
			ReplyTo     string `json:"reply_to"`
			Port        int    `json:"port"`
			UseStartTLS bool   `json:"use_starttls"`
			UseSSL      bool   `json:"use_ssl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid_request", "invalid request")
			return
		}
		req.TenantID, req.Host, req.FromEmail = strings.TrimSpace(req.TenantID), strings.TrimSpace(req.Host), strings.TrimSpace(req.FromEmail)
		if req.TenantID == "" || req.Host == "" || req.Port < 1 || req.Port > 65535 || req.FromEmail == "" {
			writeError(w, 400, "invalid_settings", "tenant, host, valid port, and from email are required")
			return
		}
		if req.UseSSL && req.UseStartTLS {
			writeError(w, 400, "invalid_settings", "choose SSL or STARTTLS, not both")
			return
		}
		stored, err := repo.GetAdministrationSMTPSettings(r.Context(), req.TenantID)
		if err != nil && err != storage.ErrNotFound {
			writeError(w, 500, "query_failed", "failed to load existing email settings")
			return
		}
		ciphertext := stored.PasswordCiphertext
		if strings.TrimSpace(req.Password) != "" {
			ciphertext, err = encryptNotificationSecret(cfg.NotificationEncryptionKey, req.Password)
			if err != nil {
				writeError(w, 503, "notification_encryption_unavailable", err.Error())
				return
			}
		}
		settings, err := repo.UpsertAdministrationSMTPSettings(r.Context(), storage.AdministrationSMTPSettings{TenantID: req.TenantID, Host: req.Host, Port: req.Port, Username: strings.TrimSpace(req.Username), PasswordCiphertext: ciphertext, UseStartTLS: req.UseStartTLS, UseSSL: req.UseSSL, FromEmail: req.FromEmail, FromName: strings.TrimSpace(req.FromName), ReplyTo: strings.TrimSpace(req.ReplyTo), UpdatedBy: p.Subject})
		if err != nil {
			writeError(w, 500, "persistence_failed", "failed to save email settings")
			return
		}
		writeJSON(w, 200, map[string]any{"settings": settings})
	})
}
func requireNotificationSuperAdmin(w http.ResponseWriter, r *http.Request, cfg RouterConfig) bool {
	p, _ := principalFromContext(r.Context())
	if cfg.Auth.Enabled && !p.SuperAdmin {
		writeError(w, 403, "super_admin_required", "super-admin access is required")
		return false
	}
	return true
}
func encryptNotificationSecret(encoded, plain string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("SIGNALOPS_NOTIFICATION_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return append(nonce, gcm.Seal(nil, nonce, []byte(plain), nil)...), nil
}
