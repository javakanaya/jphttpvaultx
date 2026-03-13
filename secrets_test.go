package jphttpvaultx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// kvv2DataPath
// ---------------------------------------------------------------------------

func TestKvv2DataPath(t *testing.T) {
	c := New(WithKVMount("static-secret"))
	got := c.kvv2DataPath("my-service/config")
	assert.Equal(t, "static-secret/data/my-service/config", got)
}

// ---------------------------------------------------------------------------
// stringField
// ---------------------------------------------------------------------------

func TestStringFieldOK(t *testing.T) {
	m := map[string]interface{}{"key": "value"}
	got, err := stringField(m, "key")
	require.NoError(t, err)
	assert.Equal(t, "value", got)
}

func TestStringFieldMissing(t *testing.T) {
	_, err := stringField(map[string]interface{}{}, "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"missing"`)
}

func TestStringFieldWrongType(t *testing.T) {
	m := map[string]interface{}{"key": 42}
	_, err := stringField(m, "key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a string")
}

// ---------------------------------------------------------------------------
// unwrapSecrets
// ---------------------------------------------------------------------------

func TestUnwrapSecretsWrapped(t *testing.T) {
	outer := map[string]interface{}{
		"secrets": map[string]interface{}{"username": "admin"},
	}
	got := unwrapSecrets(outer)
	assert.Equal(t, "admin", got["username"])
}

func TestUnwrapSecretsFlat(t *testing.T) {
	flat := map[string]interface{}{"username": "admin"}
	got := unwrapSecrets(flat)
	assert.Equal(t, "admin", got["username"])
}

func TestUnwrapSecretsSecretsNotMap(t *testing.T) {
	// "secrets" key exists but is not a map — should fall back to flat data
	data := map[string]interface{}{
		"secrets":  "not-a-map",
		"username": "admin",
	}
	got := unwrapSecrets(data)
	assert.Equal(t, "admin", got["username"])
}

// ---------------------------------------------------------------------------
// GetSecret
// ---------------------------------------------------------------------------

func TestGetSecretOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"api_key": "super-secret",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	val, err := c.GetSecret(context.Background(), "my-service/config", "api_key")
	require.NoError(t, err)
	assert.Equal(t, "super-secret", val)
}

func TestGetSecretCorrectPath(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{"k": "v"}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "my-mount", "")
	_, _ = c.GetSecret(context.Background(), "svc/cfg", "k")

	assert.Equal(t, "/v1/my-mount/data/svc/cfg", receivedPath)
}

func TestGetSecretKeyNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"other_key": "value",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.GetSecret(context.Background(), "my-service/config", "missing_key")
	require.Error(t, err)
}

func TestGetSecretWrongType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"count": 42,
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.GetSecret(context.Background(), "my-service/config", "count")
	require.Error(t, err)
}

func TestGetSecretVaultError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"errors": []string{"permission denied"},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.GetSecret(context.Background(), "my-service/config", "api_key")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetSecretMap
// ---------------------------------------------------------------------------

func TestGetSecretMapOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"k1": "v1",
			"k2": "v2",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	m, err := c.GetSecretMap(context.Background(), "my-service/config")
	require.NoError(t, err)
	assert.Equal(t, "v1", m["k1"])
	assert.Equal(t, "v2", m["k2"])
}

func TestGetSecretMapVaultError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"errors": []string{}})
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.GetSecretMap(context.Background(), "missing/secret")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetDatabaseCredentials
// ---------------------------------------------------------------------------

// wrapped layout: { "secrets": { "username": ..., "password": ... } }
func TestGetDatabaseCredentialsWrappedOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"secrets": map[string]interface{}{
				"username": "admin",
				"password": "s3cr3t",
			},
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "static-secret", "")
	creds, err := c.GetDatabaseCredentials(context.Background(), "payments-db")
	require.NoError(t, err)
	assert.Equal(t, "admin", creds.Username)
	assert.Equal(t, "s3cr3t", creds.Password)
}

// flat layout: { "username": ..., "password": ... }
func TestGetDatabaseCredentialsFlatOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"username": "admin",
			"password": "s3cr3t",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "static-secret", "")
	creds, err := c.GetDatabaseCredentials(context.Background(), "payments-db")
	require.NoError(t, err)
	assert.Equal(t, "admin", creds.Username)
	assert.Equal(t, "s3cr3t", creds.Password)
}

func TestGetDatabaseCredentialsCorrectPath(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"username": "u",
			"password": "p",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "static-secret", "")
	_, _ = c.GetDatabaseCredentials(context.Background(), "payments-db")

	assert.Equal(t, "/v1/static-secret/data/payments-db", receivedPath)
}

func TestGetDatabaseCredentialsMissingUsername(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"password": "s3cr3t",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.GetDatabaseCredentials(context.Background(), "svc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "username")
}

func TestGetDatabaseCredentialsMissingPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"username": "admin",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.GetDatabaseCredentials(context.Background(), "svc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "password")
}

func TestGetDatabaseCredentialsVaultError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"errors": []string{"internal server error"},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.GetDatabaseCredentials(context.Background(), "svc")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetThirdPartyAppCredential
// ---------------------------------------------------------------------------

// wrapped layout: { "secrets": { "email": ..., "password": ... } }
func TestGetThirdPartyAppCredentialWrappedOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"secrets": map[string]interface{}{
				"email":    "user@example.com",
				"password": "p@ssw0rd",
			},
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "static-secret", "")
	creds, err := c.GetThirdPartyAppCredential(context.Background(), "stripe")
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", creds.Email)
	assert.Equal(t, "p@ssw0rd", creds.Password)
}

// flat layout: { "email": ..., "password": ... }
func TestGetThirdPartyAppCredentialFlatOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"email":    "user@example.com",
			"password": "p@ssw0rd",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "static-secret", "")
	creds, err := c.GetThirdPartyAppCredential(context.Background(), "stripe")
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", creds.Email)
	assert.Equal(t, "p@ssw0rd", creds.Password)
}

func TestGetThirdPartyAppCredentialCorrectPath(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"email":    "e",
			"password": "p",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "static-secret", "")
	_, _ = c.GetThirdPartyAppCredential(context.Background(), "stripe")

	assert.Equal(t, "/v1/static-secret/data/stripe", receivedPath)
}

func TestGetThirdPartyAppCredentialMissingEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"password": "p@ssw0rd",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.GetThirdPartyAppCredential(context.Background(), "stripe")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email")
}

func TestGetThirdPartyAppCredentialMissingPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"email": "user@example.com",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.GetThirdPartyAppCredential(context.Background(), "stripe")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "password")
}

func TestGetThirdPartyAppCredentialVaultError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"errors": []string{"permission denied"},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.GetThirdPartyAppCredential(context.Background(), "stripe")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetSecretKey
// ---------------------------------------------------------------------------

// wrapped layout: { "secrets": { "secret_key": ... } }
func TestGetSecretKeyWrappedOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"secret_key": "my-signing-key",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "static-secret", "")
	val, err := c.GetSecretKey(context.Background(), "jwt/signing")
	require.NoError(t, err)
	assert.Equal(t, "my-signing-key", val)
}

// flat layout: { "secret_key": ... }
func TestGetSecretKeyFlatOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"secret_key": "my-signing-key",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "static-secret", "")
	val, err := c.GetSecretKey(context.Background(), "jwt/signing")
	require.NoError(t, err)
	assert.Equal(t, "my-signing-key", val)
}

func TestGetSecretKeyCorrectPath(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"secret_key": "k",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "my-mount", "")
	_, _ = c.GetSecretKey(context.Background(), "jwt/signing")

	assert.Equal(t, "/v1/my-mount/data/jwt/signing", receivedPath)
}

func TestGetSecretKeyMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vaultKVResponse(map[string]interface{}{
			"other_field": "value",
		}))
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.GetSecretKey(context.Background(), "jwt/signing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret_key")
}

func TestGetSecretKeyVaultError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"errors": []string{"permission denied"},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv, "secret", "")
	_, err := c.GetSecretKey(context.Background(), "jwt/signing")
	require.Error(t, err)
}
