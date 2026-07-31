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

func TestUTCDateDecisionBypassesDKIMSignedMessages(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantDate string
	}{
		{
			name:     "date present in h= tag preserves date",
			raw:      "DKIM-Signature: v=1; a=rsa-sha256; h=from:to:date:subject; b=...\r\nDate: Wed, 03 Jun 2026 12:34:56 -0700\r\n\r\n",
			wantDate: "Wed, 03 Jun 2026 12:34:56 -0700",
		},
		{
			name:     "date not in h= tag rewrites date to UTC",
			raw:      "DKIM-Signature: v=1; a=rsa-sha256; h=from:to:subject; b=...\r\nDate: Wed, 03 Jun 2026 12:34:56 -0700\r\n\r\n",
			wantDate: "Wed, 03 Jun 2026 19:34:56 +0000",
		},
		{
			name:     "spaces in h= tag and uppercase date",
			raw:      "dkim-signature: v=1; a=rsa-sha256; h= From : To : DATE : Subject ; b=...\r\nDate: Wed, 03 Jun 2026 12:34:56 -0700\r\n\r\n",
			wantDate: "Wed, 03 Jun 2026 12:34:56 -0700",
		},
		{
			name:     "multiple dkim signatures where second signs date",
			raw:      "DKIM-Signature: v=1; a=rsa-sha256; h=from:to; b=...\r\nDKIM-Signature: v=1; a=rsa-sha256; h=from:date; b=...\r\nDate: Wed, 03 Jun 2026 12:34:56 -0700\r\n\r\n",
			wantDate: "Wed, 03 Jun 2026 12:34:56 -0700",
		},
		{
			name:     "real-world ed25519 dkim signature with folded h= list containing date",
			raw:      "DKIM-Signature: v=1; a=ed25519-sha256; c=relaxed/relaxed; d=la.gs; s=lags2026-ed; t=1785463252; h=from:from:reply-to:reply-to:subject:subject:date:date: message-id:message-id:to:to:cc:mime-version:mime-version: content-type:content-type: content-transfer-encoding:content-transfer-encoding:list-unsubscribe: list-unsubscribe-post; bh=JfaBcpaqo4zkJomMZi9QaK6NS7UE+1g3AJAIUDTYNVQ=; b=QPYWwbuE0FAL/ry+RDQSWHWcMqn6fiLmPhjHbCpI31r2yQk0QeuZunBA79zglzfZHikMUl I/qy5BsXan65S/Ag==\r\nDate: Wed, 03 Jun 2026 12:34:56 -0700\r\n\r\n",
			wantDate: "Wed, 03 Jun 2026 12:34:56 -0700",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trx := (&testtrx.Trx{}).SetHeadersRaw([]byte(tt.raw))
			decision, err := utcDateDecision(time.Now)(context.Background(), trx)
			if err != nil {
				t.Fatal(err)
			}
			if !decision.Equal(mailfilter.Accept) {
				t.Fatalf("decision = %s, want accept", decision)
			}
			got := strings.TrimSpace(trx.Headers().Value("Date"))
			if got != tt.wantDate {
				t.Fatalf("Date = %q, want %q", got, tt.wantDate)
			}
		})
	}
}
