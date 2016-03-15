package http

import (
	"flag"
	"github.com/olomix/dynproxy/stats"
	"html/template"
	"net/http"
	"time"
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
	funcMap := template.FuncMap{
		"since": func(start time.Time) int {
			return int(time.Since(start).Seconds())
		},
	}
	var err error
	controller.tmpl, err = template.New("StatisticsTmpl").Funcs(funcMap).Parse(tmpl)
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
<html>
<body>
<h3>Stats</h3>
clientProxyNum: {{.ClientProxyNum}} <br />
proxyClientNum: {{.ProxyClientNum}} <br />
checkProxyNum: {{.CheckProxyNum}} <br />

<h3>Active requests:</h3>
<table>
<tr>
  <th>URL</th>
  <th>Cleint Addr</th>
  <th>Proxy Addr</th>
  <th>Client handler running</th>
  <th>Proxy handler running</th>
  <th>Time</th>
</tr>
{{range .Requests}}
<tr>
  <td>{{.URL}}</td>
  <td>{{.Client}}</td>
  <td>{{.Proxy}}</td>
  <td>{{.ClientHandlerRunning}}</td>
  <td>{{.ProxyHandlerRunning}}</td>
  <td>{{since .Start}}</td>
</tr>
{{end}}
</table>
</body>
</html>
`
