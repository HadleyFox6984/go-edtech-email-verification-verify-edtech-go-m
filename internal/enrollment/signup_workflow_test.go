package enrollment

import (
	"context"
	"errors"
	"testing"
	"time"
)

type recordingSender struct {
	calls int
}

func (s *recordingSender) SendVerification(context.Context, Signup) (string, error) {
	s.calls++
	return "msg_learner_42", nil
}

func TestWorkflowStart(t *testing.T) {
	now := time.Date(2026, time.August, 15, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name          string
		ready         bool
		deadline      time.Time
		wantStatus    string
		wantCalls     int
		wantReportable bool
		wantErr       error
	}{
		{name: "ready course before deadline", ready: true, deadline: now.Add(24 * time.Hour), wantStatus: "verification_sent", wantCalls: 1, wantReportable: true},
		{name: "course delivery pending", ready: false, deadline: now.Add(24 * time.Hour), wantStatus: "course_pending", wantErr: ErrCourseNotReady},
		{name: "deadline already passed", ready: true, deadline: now.Add(-time.Minute), wantStatus: "deadline_passed", wantReportable: true, wantErr: ErrDeadlinePassed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &recordingSender{}
			workflow := Workflow{Sender: sender, Now: func() time.Time { return now }}
			outcome, err := workflow.Start(context.Background(), Signup{
				ID: "signup-42", Email: "learner@example.edu", CourseName: "Ledger Controls",
				CourseReady: tt.ready, Deadline: tt.deadline, VerificationURL: "https://learn.example.edu/verify/token",
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if outcome.Status != tt.wantStatus || sender.calls != tt.wantCalls || outcome.Reportable != tt.wantReportable {
				t.Fatalf("outcome = %+v, calls = %d", outcome, sender.calls)
			}
		})
	}
}
