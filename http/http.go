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

func InitClient(caCert []byte) {
	// Load CA cert
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup Https client
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client = &http.Client{Transport: transport}
	klog.V(3).Infoln("Http client initialized")
}

func Post(jsonData []byte) (*http.Response, error) {
	url := os.Getenv("HC_API_SERVER_URL")
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return response, err
	}

	klog.V(3).Infoln("Http post request done")
	klog.V(3).Infoln("Response: ", string(body))
	return response, nil
}
