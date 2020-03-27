package transaction

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/model/metadata"
	"github.com/elastic/apm-server/transform"
)

func BenchmarkTransactionEventDecode(b *testing.B) {
	id, trType, name, result := "123", "type", "foo()", "555"
	timestampParsed := time.Date(2017, 5, 30, 18, 53, 27, 154*1e6, time.UTC)
	traceId, parentId := "0147258369012345abcdef0123456789", "abcdef0123456789"
	dropped, started, duration := 12, 6, 1.67
	name, userId, email, userIp := "jane", "abc123", "j@d.com", "127.0.0.1"
	url, referer, origUrl := "https://mypage.com", "http:mypage.com", "127.0.0.1"
	marks := map[string]interface{}{"k": "b"}
	sampled := true
	labels := model.Labels{"foo": "bar"}
	ua := "go-1.1"
	user := metadata.User{Name: &name, Email: &email, IP: net.ParseIP(userIp), Id: &userId, UserAgent: &ua}
	page := model.Page{Url: &url, Referer: &referer}
	request := model.Req{Method: "post", Socket: &model.Socket{}, Headers: http.Header{"User-Agent": []string{ua}}}
	response := model.Resp{Finished: new(bool), MinimalResp: model.MinimalResp{Headers: http.Header{"Content-Type": []string{"text/html"}}}}
	h := model.Http{Request: &request, Response: &response}
	ctxUrl := model.Url{Original: &origUrl}
	custom := model.Custom{"abc": 1}

	e := &Event{
		Id:        id,
		Type:      trType,
		Name:      &name,
		Result:    &result,
		ParentId:  &parentId,
		TraceId:   traceId,
		Duration:  duration,
		Timestamp: timestampParsed,
		Marks:     marks,
		Sampled:   &sampled,
		SpanCount: SpanCount{Dropped: &dropped, Started: &started},
		User:      &user,
		Labels:    &labels,
		Page:      &page,
		Custom:    &custom,
		Http:      &h,
		Url:       &ctxUrl,
		Client:    &model.Client{IP: net.ParseIP(userIp)},
	}

	ctx := context.Background()
	tctxt := &transform.Context{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Transform(ctx, tctxt)
	}
}
