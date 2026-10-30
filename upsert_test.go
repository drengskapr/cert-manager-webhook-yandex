package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/dns/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	dnssdk "github.com/yandex-cloud/go-sdk/services/dns/v1"
	sdkop "github.com/yandex-cloud/go-sdk/v2/pkg/operation"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
)

// fakeZones returns first from UpsertRecordSets and serves polls from poll.
type fakeZones struct {
	dnssdk.DnsZoneClient
	first *operation.Operation
	poll  sdkop.PollFunc
}

func (f fakeZones) UpsertRecordSets(context.Context, *dns.UpsertRecordSetsRequest, ...grpc.CallOption) (*dnssdk.DnsZoneUpsertRecordSetsOperation, error) {
	op, err := sdkop.NewOperation(f.first, &sdkop.Concretization{
		Poll:         f.poll,
		MetadataType: (*dns.UpsertRecordSetsMetadata)(nil),
		ResponseType: (*dns.RecordSetDiff)(nil),
	})
	if err != nil {
		return nil, err
	}
	return &dnssdk.DnsZoneUpsertRecordSetsOperation{Operation: *op}, nil
}

func testOperation(t *testing.T, done bool, result any) *operation.Operation {
	t.Helper()
	md, err := anypb.New(&dns.UpsertRecordSetsMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	op := &operation.Operation{Id: "op1", Metadata: md, Done: done}
	switch r := result.(type) {
	case *dns.RecordSetDiff:
		resp, err := anypb.New(r)
		if err != nil {
			t.Fatal(err)
		}
		op.Result = &operation.Operation_Response{Response: resp}
	case *status.Status:
		op.Result = &operation.Operation_Error{Error: r}
	}
	return op
}

func pollReturning(op *operation.Operation) sdkop.PollFunc {
	return func(context.Context, string, ...grpc.CallOption) (sdkop.YCOperation, error) {
		return op, nil
	}
}

func TestUpsertPollsUntilDone(t *testing.T) {
	sess := &dnsSession{zones: fakeZones{
		first: testOperation(t, false, nil),
		poll:  pollReturning(testOperation(t, true, &dns.RecordSetDiff{})),
	}}
	if err := sess.upsert(context.Background(), &dns.UpsertRecordSetsRequest{}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
}

func TestUpsertReturnsOperationError(t *testing.T) {
	sess := &dnsSession{zones: fakeZones{
		first: testOperation(t, false, nil),
		poll:  pollReturning(testOperation(t, true, &status.Status{Code: 7, Message: "permission denied on zone"})),
	}}
	err := sess.upsert(context.Background(), &dns.UpsertRecordSetsRequest{})
	if err == nil || !strings.Contains(err.Error(), "permission denied on zone") {
		t.Fatalf("upsert error = %v, want the operation's status", err)
	}
}

func TestUpsertStopsAtDeadline(t *testing.T) {
	sess := &dnsSession{zones: fakeZones{
		first: testOperation(t, false, nil),
		poll:  pollReturning(testOperation(t, false, nil)),
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := sess.upsert(ctx, &dns.UpsertRecordSetsRequest{}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("upsert error = %v, want context.DeadlineExceeded", err)
	}
}
