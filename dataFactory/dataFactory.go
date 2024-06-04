package dataFactory

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

type RecentLogInfo struct {
	StageTimestamp string
	AuditId        types.UID
}

const (
	DB_DRIVER   = "postgres"
	DB_USER     = "postgres"
	DB_PASSWORD = "tmax"
	DB_NAME     = "postgres"
	HOSTNAME    = "timescaledb-service.hypercloud5-system.svc"
	PORT        = "5432"
)

var dbPool *pgxpool.Pool
var ctx context.Context

func InitDBCP() {
	var err error
	ctx = context.Background()
	connStr := DB_DRIVER + "://" + DB_USER + ":" + DB_PASSWORD + "@" + HOSTNAME + ":" + PORT + "/" + DB_NAME
	dbPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		panic(err)
	}

	var greeting string
	err = dbPool.QueryRow(ctx, "SELECT 'HELLO, Timescale'").Scan(&greeting)
	if err != nil {
		panic(err)
	}

	klog.V(3).Info("Timescale DBCP initialized")
}

func GetRecentLogInfo() (string, []types.UID) {
	query := "SELECT ID, STAGETIMESTAMP FROM AUDIT ORDER BY STAGETIMESTAMP DESC LIMIT 10;"
	rows, err := dbPool.Query(ctx, query)
	if err != nil {
		klog.V(1).Infoln(err)
	}
	defer rows.Close()

	results := make([]RecentLogInfo, 0)
	for rows.Next() {
		var recentLogInfo RecentLogInfo
		err := rows.Scan(&recentLogInfo.StageTimestamp, &recentLogInfo.AuditId)
		if err != nil {
			klog.V(1).Infoln("Unable to scan ", err)
		}
		results = append(results, recentLogInfo)
	}

	if len(results) == 0 {
		klog.V(3).Infoln("No logs in Timescale DB")
		return "", make([]types.UID, 0)
	}

	recentStageTimestamp := results[0].StageTimestamp
	var recentAuditIds []types.UID
	for _, result := range results {
		if result.StageTimestamp == recentStageTimestamp {
			recentAuditIds = append(recentAuditIds, result.AuditId)
		}
	}

	return recentStageTimestamp, recentAuditIds
}
