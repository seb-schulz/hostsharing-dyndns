package main

import (
	"html/template"
	"io"
	"net"
)

const DEFAULT_TEMPLATE = `{DEFAULT}
{{- range .Subdomains }}
{{ if .IPv4 }}{{ .Subpart }}.{DOM_HOSTNAME}. {{ .TTL }} IN A {{ .IPv4 }}{{ end }}
{{ if .IPv6 }}{{ .Subpart }}.{DOM_HOSTNAME}. {{ .TTL }} IN AAAA {{ .IPv6 }}{{ end -}}
{{- end -}}`

type subdomain struct {
	Subpart string
	TTL     uint
	IPv4    *net.IP
	IPv6    *net.IP
}

type zonefile struct {
	tmpl       *template.Template
	subdomains []subdomain
}

func New(subdomains []subdomain) (*zonefile, error) {
	tmpl, err := template.New("Zonefile").Parse(DEFAULT_TEMPLATE)
	if err != nil {
		return nil, err
	}

	return &zonefile{tmpl: tmpl, subdomains: subdomains}, nil
}

func (tmpl *zonefile) Write(wr io.Writer) error {
	return tmpl.tmpl.Execute(wr, struct {
		Subdomains []subdomain
	}{Subdomains: tmpl.subdomains})
}
