# azure-audit-collector

> Azure Log Analytics에서 수집한 audit log를 hypercloud api server로 전송하는 서버

## Prerequisite

1. [Azure go sdk 인증 설정](./docs/azure.md)
2. TLS 설정
   - CA 인증서, Key가 저장된 Secret을 azure-audit-collector에 마운트 
   - CA 인증서로 서명된 인증서, key가 저장된 Secret을 Hypercloud api server에 마운트

## ENV

|variable name|value
|-|-
|`LOG_ANALYTICS_WORKSPACE_ID`|Azure Log Analytics Workspace ID. Azure Portal에서 확인 가능.
|`HC_API_SERVER_URL`|Hypercloud api server url. https://service.namespace.svc/audit 형태로 작성.
|`INTERVAL`|Log Analytics에 쿼리를 보내는 주기 (초)