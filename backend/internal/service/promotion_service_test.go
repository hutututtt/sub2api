package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromotionVisitorContextHashesCookieToken(t *testing.T) {
	secret := []byte("promotion-visitor-cookie-secret-32")
	secret = secret[:32]
	token := base64.RawURLEncoding.EncodeToString(secret)

	got := PromotionVisitorHash(WithPromotionVisitor(context.Background(), token))
	wantHash := sha256.Sum256(secret)

	require.Equal(t, hex.EncodeToString(wantHash[:]), got)
	require.NotContains(t, got, token)
}

func TestPromotionVisitorContextRejectsInvalidCookieToken(t *testing.T) {
	for _, token := range []string{"", "not-base64", base64.RawURLEncoding.EncodeToString([]byte("short"))} {
		t.Run(token, func(t *testing.T) {
			require.Empty(t, PromotionVisitorHash(WithPromotionVisitor(context.Background(), token)))
		})
	}
}

func TestValidPromotionLandingOnlyAllowsOnSitePaths(t *testing.T) {
	for _, path := range []string{"/", "/register", "/register?source=ad", "/foo/bar#section"} {
		require.True(t, ValidPromotionLanding(path), path)
	}
	for _, path := range []string{"", "https://example.com", "//example.com", "/\\\\evil.com", "/path\nnext"} {
		require.False(t, ValidPromotionLanding(path), path)
	}
}
