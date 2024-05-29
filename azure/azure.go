package azure

import (
	"context"
	"encoding/json"
	"errors"
	azto "github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/klog/v2"
	"log"
	"os"
	"time"
)

var client *azquery.LogsClient

func InitClient() {
	if client != nil {
		return
	}

	// create azure credential
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		panic(err)
	}

	// create log analytics client
	newClient, err := azquery.NewLogsClient(credential, nil)
	if err != nil {
		panic(err)
	}
	client = newClient
}

func Query(from time.Time, to time.Time) (*azquery.LogsClientQueryWorkspaceResponse, error) {
	workspaceID := os.Getenv("LOG_ANALYTICS_WORKSPACE_ID")
	klog.V(1).Infoln("workspaceID: ", workspaceID)
	query := os.Getenv("LOG_ANALYTICS_QUERY")
	if query == "" {
		query = "AKSAudit | project TimeGenerated, Level, AuditId, Stage, RequestUri, Verb, User, SourceIps, UserAgent, ObjectRef, ResponseStatus, RequestObject, ResponseObject, RequestReceivedTime, StageReceivedTime, Annotations | take 10"
	}
	klog.V(1).Infoln("Query starts: ", query)
	res, err := client.QueryWorkspace(
		context.TODO(),
		workspaceID,
		azquery.Body{
			Query:    azto.Ptr(query),
			Timespan: azto.Ptr(azquery.NewTimeInterval(from, to)),
		},
		nil,
	)

	if err != nil {
		return nil, err
	}

	if res.Error != nil {
		return nil, errors.New(res.Error.Error())
	}

	return &res, nil
}

func ToAuditEvent(row azquery.Row) *audit.Event {
	userInfo := authv1.UserInfo{}
	unmarshal(row[6], &userInfo)
	sourceIps := make([]string, 0)
	unmarshal(row[7], &sourceIps)
	objectRef := audit.ObjectReference{}
	unmarshal(row[9], &objectRef)
	responseStatus := metav1.Status{}
	unmarshal(row[10], &responseStatus)
	requestObject := runtime.Unknown{}
	unmarshal(row[11], &requestObject)
	responseObject := runtime.Unknown{}
	unmarshal(row[12], &responseObject)
	annotations := make(map[string]string)
	unmarshal(row[15], &annotations)

	return &audit.Event{
		Level:                    audit.Level(toStr(row[1])),
		AuditID:                  types.UID(toStr(row[2])),
		Stage:                    audit.Stage(toStr(row[3])),
		RequestURI:               toStr(row[4]),
		Verb:                     toStr(row[5]),
		User:                     userInfo,
		SourceIPs:                sourceIps,
		UserAgent:                row[8].(string),
		ObjectRef:                &objectRef,
		ResponseStatus:           &responseStatus,
		RequestObject:            &requestObject,
		ResponseObject:           &responseObject,
		RequestReceivedTimestamp: stringToMicroTime(row[13]),
		StageTimestamp:           stringToMicroTime(row[14]),
		Annotations:              annotations,
	}
}

func toStr(item any) string {
	if item == nil {
		return ""
	} else {
		str, ok := item.(string)
		if !ok {
			klog.V(1).Infoln("Failed casting")
			return ""
		}
		return str
	}
}

func unmarshal[T []string | map[string]string | authv1.UserInfo | audit.ObjectReference | metav1.Status | runtime.Unknown](item any, target *T) {
	if item == nil {
		return
	}
	if err := json.Unmarshal([]byte(item.(string)), target); err != nil {
		klog.V(1).Infoln("failed to unmarshal: ", err)
	}
}

func stringToMicroTime(timeItem any) metav1.MicroTime {
	if timeObj, err := time.Parse(time.RFC3339, timeItem.(string)); err != nil {
		log.Fatalf("Error parsing time: %v\n", err)
		return metav1.MicroTime{}
	} else {
		return metav1.NewMicroTime(timeObj)
	}
}
