package checkup

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sourcegraph/checkup/check/dns"
	"github.com/sourcegraph/checkup/check/exec"
	"github.com/sourcegraph/checkup/check/http"
	"github.com/sourcegraph/checkup/check/tcp"
	"github.com/sourcegraph/checkup/check/tls"
)

func checkerDecode(typeName string, config json.RawMessage) (Checker, error) {
	switch typeName {
	case "dns":
		return dns.New(config)
	case "exec":
		return exec.New(config)
	case "http":
		return http.New(config)
	case "tcp":
		return tcp.New(config)
	case "tls":
		return tls.New(config)
	default:
		return nil, errors.New(strings.Replace(errUnknownCheckerType, "%T", typeName, -1))
	}
}

func checkerType(ch interface{}) (string, error) {
	var typeName string
	switch ch.(type) {
	case dns.Checker, *dns.Checker:
		typeName = "dns"
	case exec.Checker, *exec.Checker:
		typeName = "exec"
	case http.Checker, *http.Checker:
		typeName = "http"
	case tcp.Checker, *tcp.Checker:
		typeName = "tcp"
	case tls.Checker, *tls.Checker:
		typeName = "tls"
	default:
		return "", fmt.Errorf(errUnknownCheckerType, ch)
	}
	return typeName, nil
}
