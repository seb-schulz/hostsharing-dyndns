package main

import (
	"html/template"
	"io"
	"net/netip"
)

const DEFAULT_TEMPLATE = `{DEFAULT_ZONEFILE}
{{- with .Subdomain }}
{{ if .IPv4 }}{{ .Subpart }}.{DOM_HOSTNAME}. {{ .TTL }} IN A {{ .IPv4 }}{{ end }}
{{ if .IPv6 }}{{ .Subpart }}.{DOM_HOSTNAME}. {{ .TTL }} IN AAAA {{ .IPv6 }}{{ end -}}
{{- end -}}`

type subdomain struct {
	Subpart string
	TTL     uint
	IPv4    *netip.Addr
	IPv6    *netip.Addr
}

type zonefile struct {
	tmpl      *template.Template
	subdomain subdomain
}

type zoneFileWriter interface {
	Set(s subdomain)
	Write(wr io.Writer) error
}

func newZonefile() *zonefile {
	tmpl, err := template.New("Zonefile").Parse(DEFAULT_TEMPLATE)
	if err != nil {
		panic(err)
	}

	return &zonefile{tmpl: tmpl}
}

func (tmpl *zonefile) Set(s subdomain) {
	tmpl.subdomain = s
}

func (tmpl *zonefile) Write(wr io.Writer) error {
	return tmpl.tmpl.Execute(wr, struct {
		Subdomain subdomain
	}{Subdomain: tmpl.subdomain})
}
