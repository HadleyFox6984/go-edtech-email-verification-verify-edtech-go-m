package enrollment

import (
	"context"
	"errors"
	"time"
)

type Signup struct {
	ID              string    `json:"signup_id"`
	Email           string    `json:"email"`
	CourseName      string    `json:"course_name"`
	CourseReady     bool      `json:"course_ready"`
	Deadline        time.Time `json:"deadline"`
	VerificationURL string    `json:"verification_url"`
}

type Outcome struct {
	SignupID        string    `json:"signup_id"`
	Status          string    `json:"status"`
	MessageID       string    `json:"message_id,omitempty"`
	LearnerDeadline time.Time `json:"learner_deadline"`
	Reportable      bool      `json:"reportable"`
}

type VerificationSender interface {
	SendVerification(context.Context, Signup) (string, error)
}

type Workflow struct {
	Sender VerificationSender
	Now    func() time.Time
}

var (
	ErrCourseNotReady = errors.New("course delivery is not ready")
	ErrDeadlinePassed = errors.New("learner verification deadline has passed")
)

func (w Workflow) Start(ctx context.Context, signup Signup) (Outcome, error) {
	outcome := Outcome{SignupID: signup.ID, LearnerDeadline: signup.Deadline}
	if !signup.CourseReady {
		outcome.Status = "course_pending"
		return outcome, ErrCourseNotReady
	}
	now := time.Now
	if w.Now != nil {
		now = w.Now
	}
	if !signup.Deadline.After(now()) {
		outcome.Status = "deadline_passed"
		outcome.Reportable = true
		return outcome, ErrDeadlinePassed
	}
	messageID, err := w.Sender.SendVerification(ctx, signup)
	if err != nil {
		outcome.Status = "delivery_rejected"
		outcome.Reportable = true
		return outcome, err
	}
	outcome.Status = "verification_sent"
	outcome.MessageID = messageID
	outcome.Reportable = true
	return outcome, nil
}
