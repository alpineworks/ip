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

func RawIPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := strings.Split(r.RemoteAddr, ":")[0]

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Incoming-IP", ip)
		_, _ = w.Write([]byte(ip))
	}
}

func IPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := strings.Split(r.RemoteAddr, ":")[0]

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
