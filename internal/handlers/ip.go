package handlers

import (
	_ "embed"
	"html/template"
	"net/http"
	"strings"
	"sync"
)

//go:embed templates/ip.gotmpl.html
var IPHandlerTemplateString string

var IPHandlerTemplateOnce sync.Once
var IPHandlerTemplate *template.Template

type IPHandlerTemplateData struct {
	IP      string
	Headers map[string]string
}

func init() {
	IPHandlerTemplateOnce.Do(func() {
		IPHandlerTemplate = template.Must(template.New("ip").Parse(IPHandlerTemplateString))
	})
}

// clientIP extracts the real client IP from the request, checking
// proxy headers before falling back to the direct connection address.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		// The first entry is the original client IP.
		if ip, _, _ := strings.Cut(fwd, ","); ip != "" {
			return strings.TrimSpace(ip)
		}
	}
	host, _, _ := strings.Cut(r.RemoteAddr, ":")
	return host
}

func RawIPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Incoming-IP", ip)
		_, _ = w.Write([]byte(ip))
	}
}

func IPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		w.Header().Set("X-Incoming-IP", ip)

		headers := make(map[string]string)
		for name, values := range r.Header {
			headers[name] = strings.Join(values, ", ")
		}

		data := IPHandlerTemplateData{
			IP:      ip,
			Headers: headers,
		}

		err := IPHandlerTemplate.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
