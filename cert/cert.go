package cert

import (
	"context"
	"encoding/base64"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"os"
)

var CaCert []byte

func InitCaCert() {
	err := GetCaCertFromFile()
	if err != nil {
		klog.V(1).Infoln("Failed to load ca cert file: ", err)
		klog.V(1).Infoln("Trying to load ca cert secret")

		err = GetCaCertFromSecret()
		if err != nil {
			panic(err)
		}
	}
	klog.V(3).Infoln("CA certificate loaded")
}

func GetCaCertFromFile() (err error) {
	CaCert, err = os.ReadFile(os.Getenv("CA_CERT_PATH"))
	return err
}

func GetCaCertFromSecret() (err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	namespace := "hypercloud5-system"
	secretName := "ca-secret"

	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	tlsCert := secret.Data["tls.crt"]

	CaCert, err = base64.StdEncoding.DecodeString(string(tlsCert))
	if err != nil {
		return err
	}

	return nil
}
