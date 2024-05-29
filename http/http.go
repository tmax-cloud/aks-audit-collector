package http

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"io"
	"k8s.io/klog/v2"
	"net/http"
	"os"
)

var client *http.Client

func init() {
	initHttpClient()
}

func initHttpClient() {
	// Load CA cert
	caCert, err := os.ReadFile(os.Getenv("CA_CERT_PATH"))
	if err != nil {
		klog.V(1).Infoln(err)
		return
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup Https client
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client = &http.Client{Transport: transport}
}

func Post(jsonData []byte) {
	url := os.Getenv("HC_API_SERVER_URL")
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		klog.V(1).Infoln(err)
		return
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		klog.V(1).Infoln(err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		klog.V(1).Infoln(err)
		return
	}

	klog.V(3).Infoln("Http post request done")
	klog.V(3).Infoln("Response: ", string(body))
}
