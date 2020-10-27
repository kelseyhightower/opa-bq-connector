package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"

	"github.com/kelseyhightower/run"

	"cloud.google.com/go/bigquery"
)

var (
	dataset string
	table   string
)

type DecisionLog struct {
	Bundles     json.RawMessage `json:"bundles"`
	DecisionID  string          `json:"decision_id"`
	Input       json.RawMessage `json:"input"`
	Labels      json.RawMessage `json:"labels"`
	Path        string          `json:"path"`
	RequestedBy string          `json:"requested_by"`
	Result      json.RawMessage `json:"result"`
	Timestamp   string          `json:"timestamp"`
}

type DecisionLogRow struct {
	Bundles     string `bigquery:"bundles"`
	DecisionID  string `bigquery:"decision_id"`
	Input       string `bigquery:"input"`
	Labels      string `bigquery:"labels"`
	Path        string `bigquery:"path"`
	RequestedBy string `bigquery:"requested_by"`
	Result      string `bigquery:"result"`
	Timestamp   string `bigquery:"timestamp"`
}

func main() {
	flag.StringVar(&dataset, "dataset", "opa", "The BigQuery dataset to write to")
	flag.StringVar(&table, "table", "decision_logs", "The BigQuery table to write to")
	flag.Parse()

	run.Notice("Starting opa bigquery connector service...")

	projectID, err := run.ProjectID()
	if err != nil {
		run.Fatal(err)
	}

	run.Notice("Creating bq client...")
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		run.Fatal(err)
	}

	inserter := client.Dataset(dataset).Table(table).Inserter()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		gzipReader, err := gzip.NewReader(r.Body)
		if err != nil {
			run.Error(r, err)
			http.Error(w, err.Error(), 500)
			return
		}

		data, err := ioutil.ReadAll(gzipReader)
		if err != nil {
			run.Error(r, err)
			http.Error(w, err.Error(), 500)
			return
		}

		defer r.Body.Close()
		defer gzipReader.Close()

		var logs []DecisionLog
		err = json.Unmarshal(data, &logs)
		if err != nil {
			run.Error(r, err)
			http.Error(w, err.Error(), 500)
			return
		}

		logRows := make([]DecisionLogRow, 0)
		for _, l := range logs {
			lr := DecisionLogRow{
				Bundles:     string(l.Bundles),
				DecisionID:  l.DecisionID,
				Input:       string(l.Input),
				Labels:      string(l.Labels),
				Path:        l.Path,
				RequestedBy: l.RequestedBy,
				Result:      string(l.Result),
				Timestamp:   l.Timestamp,
			}

			logRows = append(logRows, lr)
		}

		ctx := context.Background()
		err = inserter.Put(ctx, logRows)
		if err != nil {
			run.Error(r, err)
			http.Error(w, err.Error(), 500)
			return
		}
	})

	if err := run.ListenAndServe(nil); err != http.ErrServerClosed {
		run.Fatal(err)
	}
}
