package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func parseCIDR(s string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return ipNet
}

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestTrustedSubnet(t *testing.T) {
	subnet := parseCIDR("192.168.1.0/24")

	tests := []struct {
		name       string
		subnet     *net.IPNet
		remoteAddr string
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "nil subnet — всегда 403",
			subnet:     nil,
			remoteAddr: "192.168.1.10:1234",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "X-Real-IP в подсети — 200",
			subnet:     subnet,
			remoteAddr: "10.0.0.1:1234",
			headers:    map[string]string{"X-Real-IP": "192.168.1.42"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "X-Real-IP вне подсети — 403",
			subnet:     subnet,
			remoteAddr: "192.168.1.10:1234",
			headers:    map[string]string{"X-Real-IP": "10.0.0.1"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "X-Forwarded-For первый адрес в подсети — 200",
			subnet:     subnet,
			remoteAddr: "10.0.0.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.5, 10.0.0.1"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "X-Forwarded-For первый адрес вне подсети — 403",
			subnet:     subnet,
			remoteAddr: "192.168.1.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "8.8.8.8, 192.168.1.1"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "X-Real-IP приоритетнее X-Forwarded-For",
			subnet:     subnet,
			remoteAddr: "10.0.0.1:1234",
			headers: map[string]string{
				"X-Real-IP":       "192.168.1.99",
				"X-Forwarded-For": "8.8.8.8",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "RemoteAddr в подсети, заголовки отсутствуют — 200",
			subnet:     subnet,
			remoteAddr: "192.168.1.7:5555",
			wantStatus: http.StatusOK,
		},
		{
			name:       "RemoteAddr вне подсети, заголовки отсутствуют — 403",
			subnet:     subnet,
			remoteAddr: "172.16.0.1:5555",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "невалидный X-Real-IP, фолбек на RemoteAddr в подсети — 200",
			subnet:     subnet,
			remoteAddr: "192.168.1.3:1234",
			headers:    map[string]string{"X-Real-IP": "not-an-ip"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "все источники невалидны — 403",
			subnet:     subnet,
			remoteAddr: "invalid",
			headers:    map[string]string{"X-Real-IP": "bad", "X-Forwarded-For": "also-bad"},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			rr := httptest.NewRecorder()
			TrustedSubnet(tt.subnet)(okHandler).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("статус %d, ожидался %d", rr.Code, tt.wantStatus)
			}
		})
	}
}
