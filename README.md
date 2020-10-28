# opa-bq-connector

The OPA BigQuery connector writes [decision logs](https://www.openpolicyagent.org/docs/latest/management/#decision-logs) into [BigQuery](https://cloud.google.com/bigquery).

## Usage

```
Usage of opa-bq-connector:
  -dataset string
        The BigQuery dataset to write to (default "opa")
  -table string
        The BigQuery table to write to (default "decision_logs")
```

## Tutorial

```
PROJECT_ID=$(gcloud config get-value project)
```

```
gcloud iam service-accounts create opa-bq-connector
```

```
bq --location=US mk -d \
  --default_table_expiration 84000 \
  --description "Open Policy Agent dataset." \
  "${PROJECT_ID}:opa"
```

```
bq mk \
  --description "OPA decision logs" \
  --table "${PROJECT_ID}:opa.decision_logs" \
  'bundles:STRING,decision_id:STRING,input:STRING,labels:STRING,path:STRING,requested_by:STRING,result:STRING,timestamp:STRING'
```

```
SERVICE_ACCOUNT="opa-bq-connector@${PROJECT_ID}.iam.gserviceaccount.com"
```

```
bq show --format=prettyjson hightowerlabs:opa | \
  jq --arg sa $SERVICE_ACCOUNT \
    '.access += [{"role": "READER", "userByEmail": $sa}]' \
  > dataset-policy.json
```

```
bq update \
  --source dataset-policy.json \ 
  "$PROJECT_ID:opa"
```

```
cat <<EOF > table-policy.json
{
  "bindings": [
    {
      "members": [
        "serviceAccount:${SERVICE_ACCOUNT}"
      ],
      "role": "roles/bigquery.dataEditor"
    }
  ]
}
EOF
```
```
bq set-iam-policy \
   "${PROJECT_ID}:opa.decision_logs" \
   table-policy.json
```

```
gcloud beta run deploy opa-bq-connector \
  --concurrency 80 \
  --cpu 1 \
  --image "gcr.io/${PROJECT_ID}/opa-bq-connector:0.0.1" \
  --memory '1G' \
  --min-instances 1 \
  --no-allow-unauthenticated \
  --platform managed \
  --port 8080 \
  --region us-west1 \
  --service-account ${SERVICE_ACCOUNT} \
  --timeout 300
```

```
URL=$(gcloud run services describe opa-bq-connector \
  --platform managed \
  --region us-west1 \
  --format json | \
  jq -r '.status.url')
```

```
curl -X POST -i $URL/logs \
  -H 'Content-Encoding: gzip' \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
  --data-binary @decision.log.gz
```
