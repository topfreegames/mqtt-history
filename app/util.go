package app

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/topfreegames/mqtt-history/es"
	"gopkg.in/olivere/elastic.v5"
)

func getLimitedIndexString() string {
	return fmt.Sprintf(
		"chat-%s*,chat-%s*",
		time.Now().Local().Format("2006-01-02"),
		time.Now().Local().Add(-24*time.Hour).Format("2006-01-02"),
	)
}

func getPastIndexString() string {
	var buffer bytes.Buffer
	t := time.Now().Local().Add(time.Duration(2*-24) * time.Hour).Format("2006-01-02")
	buffer.WriteString(fmt.Sprintf("chat-%s*", t))
	for cnt := 3; cnt <= 30; cnt++ {
		t := time.Now().Local().Add(time.Duration(cnt*-24) * time.Hour).Format("2006-01-02")
		buffer.WriteString(fmt.Sprintf(",chat-%s*", t))
	}
	return buffer.String()
}

// DoESQuery does a query
func DoESQuery(index string, boolQuery *elastic.BoolQuery, from, limit int) (*elastic.SearchResult, error) {
	esclient := es.GetESClient()
	return esclient.Search().Index(getLimitedIndexString()).Query(boolQuery).
		Sort("timestamp", false).From(from).Size(limit).Do(context.TODO())
}
