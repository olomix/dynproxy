package http

import (
	"flag"
	"fmt"
	"github.com/olomix/dynproxy/stats"
	"net/http"
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
	grs *stats.GoRoutineStats
}

func ListenAndServe(grs *stats.GoRoutineStats) {
	var controller *HttpController = new(HttpController)
	controller.grs = grs
	go http.ListenAndServe(controlAddress, controller)
}

func (c *HttpController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `clientProxyNum: %v
proxyClientNum: %v
checkProxyNum: %v
	`,
		c.grs.GetClientProxy(),
		c.grs.GetProxyClient(),
		c.grs.GetCheckProxy(),
	)
}
