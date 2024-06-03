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
	"os"
	"time"
)

type AzQueryResult struct {
	RecentStageTimestamp string
	RecentAuditIds       []types.UID
	AuditEventList       audit.EventList
}

var client *azquery.LogsClient

// InitClient makes Azure Log Analytics client.
// To set proper credential, see https://learn.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication?tabs=bash#option-1-define-environment-variables
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

// Query does query to Azure Log Analytics.
func Query(query string) (*azquery.LogsClientQueryWorkspaceResponse, error) {
	klog.V(3).Info("Start query to Azure Log Analytics")
	workspaceID := os.Getenv("LOG_ANALYTICS_WORKSPACE_ID")
	res, err := client.QueryWorkspace(
		context.TODO(),
		workspaceID,
		azquery.Body{
			Query: azto.Ptr(query),
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

// GetResult makes AzQueryResult from LogsClientQueryWorkspaceResponse
// and removes any of the LogsClientQueryWorkspaceResponse matches lastTimed, lastAuditIds.
func GetResult(response *azquery.LogsClientQueryWorkspaceResponse, lastTimeGenerated string, lastAuditIds []types.UID) (*AzQueryResult, error) {
	// init azQueryResult
	table := (*response).Tables[0]
	azQueryResult := AzQueryResult{
		RecentStageTimestamp: table.Rows[0][14].(string),
		RecentAuditIds:       make([]types.UID, 0),
	}

	// search duplicate log
	var removeIdxList []int
	for idx, row := range table.Rows {
		if row[14] == lastTimeGenerated {
			for _, auditId := range lastAuditIds {
				if row[2] == string(auditId) {
					removeIdxList = append(removeIdxList, idx)
				}
			}
		}
		break
	}

	// remove duplicate log
	for _, idx := range removeIdxList {
		table.Rows = append(table.Rows[:idx], table.Rows[idx+1:]...)
	}

	// rows to azQueryResult
	eventList := audit.EventList{
		Items: make([]audit.Event, len(table.Rows)),
	}
	for idx, row := range table.Rows {
		event := rowToAuditEvent(row)
		if row[14] == azQueryResult.RecentStageTimestamp {
			azQueryResult.RecentAuditIds = append(azQueryResult.RecentAuditIds, event.AuditID)
		}
		eventList.Items[idx] = *event
	}
	azQueryResult.AuditEventList = eventList

	return &azQueryResult, nil
}

func rowToAuditEvent(row azquery.Row) *audit.Event {
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
		klog.V(1).Infoln("Error parsing time: ", err)
		return metav1.MicroTime{}
	} else {
		return metav1.NewMicroTime(timeObj)
	}
}
