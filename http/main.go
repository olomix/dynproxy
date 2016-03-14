package http

import (
	"flag"
	"github.com/olomix/dynproxy/stats"
	"net/http"
	"text/template"
)

var controlAddress string

func init() {
	flag.StringVar(
		&controlAddress,
		"httpaddr",
		":4138",
		"Address to listen control http connection on",
	)
}

type HttpController struct {
	grs  *stats.GoRoutineStats
	tmpl *template.Template
}

func ListenAndServe(grs *stats.GoRoutineStats) {
	var controller *HttpController = new(HttpController)
	controller.grs = grs
	var err error
	controller.tmpl, err = template.New("StatisticsTmpl").Parse(tmpl)
	if err != nil {
		panic(err)
	}
	go http.ListenAndServe(controlAddress, controller)
}

func (c *HttpController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.tmpl.Execute(w, struct {
		ClientProxyNum uint64
		ProxyClientNum uint64
		CheckProxyNum  uint64
		Requests       []stats.Request
	}{
		c.grs.GetClientProxy(),
		c.grs.GetProxyClient(),
		c.grs.GetCheckProxy(),
		c.grs.ActiveRequests(),
	})
}

const tmpl = `
clientProxyNum: {{.ClientProxyNum}}
proxyClientNum: {{.ProxyClientNum}}
checkProxyNum: {{.CheckProxyNum}}

Active requests:
{{range .Requests}}
{{- .}}
{{end}}
`
