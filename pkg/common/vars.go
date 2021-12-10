package common

import (
	"io/ioutil"
	"strings"
)

var (
	NodeSelector            = map[string]string{"kubernetes.io/os": "linux"}
	FalconShellCommand      = []string{"/bin/sh"}
	OrigDSConfVersion       = "0"
	FalconOperatorNamespace = "falcon-operator"
	FalconInjectorCommand   = []string{"injector"}
	Validity                = 3650
	CA                      = GenCA(FalconOperatorNamespace, Validity)
)

func init() {
	nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		ns := strings.TrimSpace(string(nsBytes))
		FalconOperatorNamespace = ns
	}
}
