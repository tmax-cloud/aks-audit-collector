package main

import (
	"encoding/json"
	"github.com/tmax-cloud/azure-collector/azure"
	"github.com/tmax-cloud/azure-collector/cert"
	"github.com/tmax-cloud/azure-collector/dataFactory"
	"github.com/tmax-cloud/azure-collector/http"
	"github.com/tmax-cloud/azure-collector/logger"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"os"
	"strconv"
	"time"
)

var (
	recentStageTimestamp string
	recentAuditIds       []types.UID
	query                string
	interval             int
)

func init() {
	logger.InitLogging()
	azure.InitClient()
	dataFactory.InitDBCP()
	cert.InitCaCert()
	http.InitClient(cert.CaCert)
	initVariable()
}

func main() {
	for {
		sendAuditLog()
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func sendAuditLog() {
	timeClause := ""
	if recentStageTimestamp != "" {
		timeClause = "| where StageReceivedTime >= todatetime(\"" + recentStageTimestamp + "\")"
	}

	// Query to Azure Log Analytics
	azQueryResponse, err := azure.Query(query + timeClause)
	if err != nil {
		klog.V(1).Infoln(err)
		return
	}

	klog.V(3).Infof("Responded %d rows\n", len(azQueryResponse.Tables[0].Rows))

	// Process query response
	resultToServe, err := azure.GetResult(azQueryResponse, recentStageTimestamp, recentAuditIds)
	if err != nil {
		klog.V(1).Infoln(err)
		return
	}
	if len(resultToServe.AuditEventList.Items) == 0 {
		klog.V(3).Infoln("Nothing to serve")
		return
	}

	// Marshal to JSON
	eventListJson, err := json.Marshal(resultToServe.AuditEventList)
	if err != nil {
		klog.V(1).Infoln(err)
		return
	}

	// Send Http POST request
	httpRes, err := http.Post(eventListJson)
	if err != nil {
		klog.V(1).Infoln(err.Error())
		return
	}

	httpResponse := *httpRes
	if httpResponse.StatusCode/2 == 100 {
		recentStageTimestamp = resultToServe.RecentStageTimestamp
		recentAuditIds = resultToServe.RecentAuditIds
		klog.V(3).Infoln("Audit log is sent successfully")
	} else {
		klog.V(1).Infoln(httpResponse.Status)
	}
}

func initVariable() {
	// init interval
	envInterval, err := strconv.Atoi(os.Getenv("INTERVAL"))
	if err != nil {
		interval = 20
		klog.V(1).Infof("Failed to load env INTERVAL. Set interval to %d seconds\n", interval)
	} else {
		interval = envInterval
	}

	// init query
	query = os.Getenv("LOG_ANALYTICS_QUERY")
	if query == "" {
		query = `AKSAudit
| where Stage !in ("ResponseStarted", "RequestReceived")
| where not(User["groups"] has_any (dynamic(["system:serviceaccounts:hypercloud5-system", "system:nodes", "system:masters", "system:serviceaccounts:kube-system", "system:serviceaccounts:monitoring"])))
| where not(User["username"] has_any (dynamic(["system:serviceaccount:hypercloud5-system:hypercloud5-admin", "system:kube-controller-manager", "system:kube-scheduler", "system:apiserver"])))
| where Verb !in ("watch", "get", "list")
| where Level == "Metadata" and ObjectRef["apiGroup"] in ("",  "admissionregistration.k8s.io",  "apiextensions.k8s.io",  "apiregistration.k8s.io",  "apps",  "authentication.istio.io",  "autoscaling",  "batch",  "cdi.kubevirt.io",  "ceph.rook.io",  "cluster.x-k8s.io",  "config.istio.io",  "core.kubefed.io",  "extensions",  "kubevirt.io",  "networking.istio.io",  "policy",  "rbac.authorization.k8s.io",  "rbac.istio.io",  "security.istio.io",  "servicecatalog.k8s.io",  "storage.k8s.io",  "tekton.dev",  "tmax.io",  "claim.tmax.io",  "cluster.tmax.io",  "types.kubefed.io")
| project TimeGenerated, Level, AuditId, Stage, RequestUri, Verb, User, SourceIps, UserAgent, ObjectRef, ResponseStatus, RequestObject, ResponseObject, RequestReceivedTime, StageReceivedTime, Annotations
| order by StageReceivedTime desc
| where TimeGenerated <= now()`
	}

	// init recent log info
	recentStageTimestamp, recentAuditIds = dataFactory.GetRecentLogInfo()
}
