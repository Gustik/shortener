package main

import (
	"crypto/x509"
	"net"
	"testing"
	"time"
)

func TestGenerateSelfSignedTLS(t *testing.T) {
	tlsCfg, err := generateSelfSignedTLS()
	if err != nil {
		t.Fatalf("generateSelfSignedTLS() вернул ошибку: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("ожидался *tls.Config, получили nil")
	}
	if len(tlsCfg.Certificates) != 1 {
		t.Fatalf("ожидался 1 сертификат, получили %d", len(tlsCfg.Certificates))
	}

	cert, err := x509.ParseCertificate(tlsCfg.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatalf("не удалось разобрать сертификат: %v", err)
	}

	t.Run("срок действия", func(t *testing.T) {
		now := time.Now()
		if cert.NotBefore.After(now) {
			t.Error("NotBefore позже текущего времени")
		}
		if cert.NotAfter.Before(now.AddDate(0, 11, 0)) {
			t.Error("срок действия меньше 11 месяцев")
		}
	})

	t.Run("DNS имена", func(t *testing.T) {
		found := false
		for _, name := range cert.DNSNames {
			if name == "localhost" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("localhost не найден в DNSNames: %v", cert.DNSNames)
		}
	})

	t.Run("IP адреса", func(t *testing.T) {
		loopback := net.ParseIP("127.0.0.1")
		found := false
		for _, ip := range cert.IPAddresses {
			if ip.Equal(loopback) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("127.0.0.1 не найден в IPAddresses: %v", cert.IPAddresses)
		}
	})

	t.Run("KeyUsage", func(t *testing.T) {
		if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
			t.Error("KeyUsageDigitalSignature не выставлен")
		}
		if cert.KeyUsage&x509.KeyUsageKeyEncipherment == 0 {
			t.Error("KeyUsageKeyEncipherment не выставлен")
		}
	})

	t.Run("ExtKeyUsage", func(t *testing.T) {
		found := false
		for _, eku := range cert.ExtKeyUsage {
			if eku == x509.ExtKeyUsageServerAuth {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ExtKeyUsageServerAuth не найден: %v", cert.ExtKeyUsage)
		}
	})
}
