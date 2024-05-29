package http

import (
	"bytes"
	"io"
	"k8s.io/klog/v2"
	"net/http"
)

func Post(jsonData []byte) {
	url := "http://localhost:38080/post"
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		klog.V(1).Infoln(err)
		return
	}
	request.Header.Set("Content-Type", "application/json")

	client := http.Client{}
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
