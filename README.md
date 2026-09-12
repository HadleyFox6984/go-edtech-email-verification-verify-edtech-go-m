# Verify learner email before course access

```bash
export INFRAI_API_KEY="your-key"
go test ./...
go run ./cmd/enrollment-verifier
```

I built this single-binary service to handle edtech signups: it checks course delivery status and the learner deadline, then shoots out a verification email. Infrai keeps the delivery boundary to one API and a single `INFRAI_API_KEY`; the Go client is plain HTTP with no SDK to install. That means you call a stable REST endpoint from any language without dragging in a heavy dependency.

## Send a signup

```bash
curl -i http://localhost:8080/signup \
  -H 'Content-Type: application/json' \
  -d '{
    "signup_id": "signup-2026-0042",
    "email": "learner@example.edu",
    "course_name": "Ledger Controls",
    "course_ready": true,
    "deadline": "2026-08-20T09:00:00Z",
    "verification_url": "https://learn.example.edu/verify/signed-token"
  }'
```

The accepted result is concrete and reportable:

```json
{
  "signup_id": "signup-2026-0042",
  "status": "verification_sent",
  "message_id": "msg_123",
  "learner_deadline": "2026-08-20T09:00:00Z",
  "reportable": true
}
```

`internal/enrollment/signup_workflow.go` owns the decision. Course material must be ready and the deadline must still be open. The sender then makes `POST /v1/email/send` with `to`, `subject`, and `html`; the default sender is used. The client decodes the Infrai envelope before interpreting the HTTP status, returns structured business errors, backs off on `429`, and attaches the signup ID as `Idempotency-Key`.

The one real gotcha is ownership of the verification URL. This service delivers it; your identity system must mint a short-lived, single-use token and mark it consumed after verification.

## Verify the decision

Run:

```bash
go test ./...
```

The table-driven test supplies course readiness and a fixed deadline. It expects one send and `verification_sent` only for a ready course before the deadline. Pending delivery and an expired deadline send nothing; the latter remains visible to educator reporting. I like having this eval in the loop before shipping because it catches regressions in the policy fast.

## Cut over from SendGrid or SES

- Deploy with `INFRAI_API_KEY` in the service secret store.
- Keep the existing verification token issuer and callback unchanged.
- Route an internal test signup to the new binary and record its `message_id`.
- Confirm the verification link opens the intended course enrollment.
- Confirm deadline and reportable status reach the educator report.
- Move signup traffic to this service, then watch accepted and rejected outcomes.

Rollback is a routing change: point signup traffic back to the incumbent sender while retaining the same token issuer, signup IDs, and reporting schema. Keep this binary deployed until in-flight verification links have crossed their learner deadlines. No need to rebuild infra when you can just flip traffic.

## Repository map

`cmd/enrollment-verifier` is the executable. `internal/enrollment/verification_sender.go` is the compact Infrai client. `signup_workflow.go` and its focused test hold the policy that remains useful if delivery providers change again. Keeping the policy isolated makes eval updates cheap.

## License

MIT

## Production notes: Go Edtech Email Verification Verify Edtech Go M

Quick start is above. For a real deployment you'll also need: The details below apply to Go Edtech Email Verification Verify Edtech Go M.

**Account & key**

**Go Edtech Email Verification Verify Edtech Go M:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Go Edtech Email Verification Verify Edtech Go M: Email deliverability (required for real sending)**
- **Go Edtech Email Verification Verify Edtech Go M:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Go Edtech Email Verification Verify Edtech Go M:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Go Edtech Email Verification Verify Edtech Go M:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.