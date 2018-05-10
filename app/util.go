package app

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/topfreegames/mqtt-history/es"
	"gopkg.in/olivere/elastic.v5"
)

func getLimitedIndexString(days int) string {
	var buffer bytes.Buffer
	t := time.Now().Local().Format("2006-01-02")
	buffer.WriteString(fmt.Sprintf("chat-%s*", t))
	for cnt := 1; cnt <= days; cnt++ {
		t := time.Now().Local().Add(time.Duration(cnt*-24) * time.Hour).Format("2006-01-02")
		buffer.WriteString(fmt.Sprintf(",chat-%s*", t))
	}
	return buffer.String()
}

// DoESQuery does a query
func DoESQuery(ctx context.Context, numberOfDaysToSearch int, boolQuery *elastic.BoolQuery, from, limit int) (*elastic.SearchResult, error) {
	esclient := es.GetESClient()
	return esclient.Search().Index(getLimitedIndexString(numberOfDaysToSearch)).Query(boolQuery).
		Sort("timestamp", false).From(from).Size(limit).Do(ctx)
}

// IsNotFoundError returns true if error is of not found type
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "not found")
}
