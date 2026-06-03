package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/d--j/go-milter/mailfilter"
	"github.com/d--j/go-milter/mailfilter/testtrx"
)

func TestUTCDateDecisionRewritesDateToUTC(t *testing.T) {
	trx := (&testtrx.Trx{}).SetHeadersRaw([]byte("Date: Wed, 03 Jun 2026 12:34:56 -0700\r\n\r\n"))

	decision, err := utcDateDecision(time.Now)(context.Background(), trx)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Equal(mailfilter.Accept) {
		t.Fatalf("decision = %s, want accept", decision)
	}

	got := strings.TrimSpace(trx.Headers().Value("Date"))
	want := "Wed, 03 Jun 2026 19:34:56 +0000"
	if got != want {
		t.Fatalf("Date = %q, want %q", got, want)
	}
}

func TestUTCDateDecisionAddsMissingDate(t *testing.T) {
	trx := (&testtrx.Trx{}).SetHeadersRaw([]byte("Subject: test\r\n\r\n"))
	now := func() time.Time {
		return time.Date(2026, time.June, 3, 12, 34, 56, 0, time.FixedZone("PDT", -7*60*60))
	}

	_, err := utcDateDecision(now)(context.Background(), trx)
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(trx.Headers().Value("Date"))
	want := "Wed, 03 Jun 2026 19:34:56 +0000"
	if got != want {
		t.Fatalf("Date = %q, want %q", got, want)
	}
}

func TestUTCDateDecisionReplacesMalformedDate(t *testing.T) {
	trx := (&testtrx.Trx{}).SetHeadersRaw([]byte("Date: not a date\r\n\r\n"))
	now := func() time.Time {
		return time.Date(2026, time.June, 3, 12, 34, 56, 0, time.FixedZone("PDT", -7*60*60))
	}

	_, err := utcDateDecision(now)(context.Background(), trx)
	if err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(trx.Headers().Value("Date"))
	want := "Wed, 03 Jun 2026 19:34:56 +0000"
	if got != want {
		t.Fatalf("Date = %q, want %q", got, want)
	}
}
