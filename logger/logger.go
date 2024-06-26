package logger

import (
	"flag"
	azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/robfig/cron"
	"io"
	"io/ioutil"
	"k8s.io/klog/v2"
	"os"
	"strings"
	"time"
)

func InitLogging() {
	// init klog
	var logLevel string
	flag.StringVar(&logLevel, "log-level", "INFO", "Log Level; TRACE, DEBUG, INFO, WARN, ERROR, FATAL")
	klog.InitFlags(nil)
	klog.Infoln("LOG_LEVEL = ", logLevel)

	if strings.EqualFold(logLevel, "TRACE") {
		logLevel = "5"
	} else if strings.EqualFold(logLevel, "DEBUG") {
		logLevel = "4"
	} else if strings.EqualFold(logLevel, "INFO") {
		logLevel = "3"
	} else if strings.EqualFold(logLevel, "WARN") {
		logLevel = "2"
	} else if strings.EqualFold(logLevel, "ERROR") {
		logLevel = "1"
	} else if strings.EqualFold(logLevel, "FATAL") {
		logLevel = "0"
	} else {
		klog.Infoln("Unknown log-level paramater. Set to default level INFO")
		logLevel = "3"
	}
	flag.Set("v", logLevel)
	flag.Parse()

	if _, err := os.Stat("./logs"); os.IsNotExist(err) {
		os.Mkdir("./logs", os.ModeDir)
	}

	file, err := os.OpenFile("./logs/azure-collector.log", os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		klog.V(1).Info(err, "Error Open", "./logs/azure-collector.log")
	}

	w := io.MultiWriter(file, os.Stdout)
	klog.SetOutput(w)

	// Logging Cron Job
	cronJob_Logging := cron.New()
	cronJob_Logging.AddFunc("1 0 0 * * ?", func() {
		input, err := ioutil.ReadFile("./logs/azure-collector.log")
		if err != nil {
			klog.V(1).Info(err)
			return
		}
		err = ioutil.WriteFile("./logs/azure-collector"+time.Now().Format("2006-01-02")+".log", input, 0644)
		if err != nil {
			klog.V(1).Info(err, "Error creating", "./logs/azure-collector.log")
			return
		}
		klog.V(3).Info("Log BackUp Success")
		os.Truncate("./logs/azure-collector.log", 0)
		file.Seek(0, io.SeekStart)
	})
	cronJob_Logging.Start()
	klog.V(1).Info("Logger initialized")

	// init az credential log
	azlog.SetListener(func(event azlog.Event, s string) {
		klog.V(3).Infoln(s)
	})
	azlog.SetEvents(azidentity.EventAuthentication)
}
