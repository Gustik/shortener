package middleware

import (
	"net"
	"net/http"
	"strings"
)


// TrustedSubnet возвращает middleware, разрешающий доступ только для IP-адресов
// из указанной подсети CIDR. При subnet == nil любой запрос получает 403.
func TrustedSubnet(subnet *net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if subnet == nil {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			ip := resolveIP(r)
			if ip == nil || !subnet.Contains(ip) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// resolveIP извлекает IP клиента из запроса.
// Приоритет: X-Real-IP → X-Forwarded-For → RemoteAddr.
// Последний вариант покрывает прямое подключение без прокси.
func resolveIP(r *http.Request) net.IP {
	if ip := net.ParseIP(r.Header.Get("X-Real-IP")); ip != nil {
		return ip
	}

	// X-Forwarded-For может содержать цепочку адресов: "client, proxy1, proxy2"
	// Нас интересует только первый — это адрес исходного клиента.
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if ip := net.ParseIP(strings.TrimSpace(strings.SplitN(forwarded, ",", 2)[0])); ip != nil {
			return ip
		}
	}

	// Прямое подключение без прокси: адрес в формате host:port.
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip
		}
	}

	return nil
}
