package anet2

import "net/http"

func temp() {
	my_req := &http.Request{
		Method: http.MethodConnect,
		Proto:  "webtransport",
	}
}
