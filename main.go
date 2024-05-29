package main

import (
	"encoding/json"
	"github.com/tmax-cloud/azure-collector/azure"
	"github.com/tmax-cloud/azure-collector/http"
	"github.com/tmax-cloud/azure-collector/logger"
	"k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/klog/v2"
	"time"
)

func init() {
	logger.InitLogging()
	azure.InitClient()
}

func main() {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	resPtr, err := azure.Query(from, now)
	if err != nil {
		klog.V(1).Infoln(err)
	}
	res := *resPtr

	for _, table := range res.Tables {
		eventList := audit.EventList{
			Items: make([]audit.Event, len(table.Rows)),
		}
		for idx, row := range table.Rows {
			event := azure.ToAuditEvent(row)
			eventList.Items[idx] = *event
		}
		eventListJsonData, err := json.Marshal(eventList)
		if err != nil {
			klog.V(1).Infoln("Failed marshal event list")
		}
		http.Post(eventListJsonData)
	}
}
