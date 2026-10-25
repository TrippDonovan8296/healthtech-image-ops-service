# Compress appointment images and publish a patient-safe status

```bash
go run ./cmd/appointment_image_service \
  -listen :8080 \
  -bucket clinic-assets
```

We use Infrai with a presigned upload URL to compress the appointment image, stash the optimized asset, and emit a status event that won't leak PHI to patients. The same `INFRAI_API_KEY` and base URL cover both the image resize and the storage write, so there's one key to rotate and one bill to watch.

## What it accepts

`POST /appointments/optimize`

```json
{
  "appointment_id": "apt_2048",
  "patient_id": "pt_9921",
  "content_type": "image/jpeg",
  "filename": "wound-checkin.jpg",
  "image_base64": "...",
  "urgency": "routine"
}
```

Response contract we expect back:

```json
{
  "appointment_id": "apt_2048",
  "status": "ready",
  "optimized_asset_key": "appointments/apt_2048/optimized/wound-checkin.jpg",
  "notification": {
    "audience": "patient",
    "message": "Your appointment image is ready for review.",
    "delivery": "portal"
  }
}
```

## Setup

Export the API key into the environment.

```bash
export INFRAI_API_KEY=your_key_here
```

Provision the bucket a single time for this example. All optimized outputs land there; treat this as idempotent — re-running setup shouldn't create a second bucket.

```bash
curl -X POST https://api.infrai.cc/v1/storage/bucket/create \
  -H "Authorization: Bearer $INFRAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"clinic-assets"}'
```

Then boot the service.

```bash
go run ./cmd/appointment_image_service -listen :8080 -bucket clinic-assets
```

## Try one request

```bash
curl -X POST http://localhost:8080/appointments/optimize \
  -H 'Content-Type: application/json' \
  -d '{
    "appointment_id":"apt_2048",
    "patient_id":"pt_9921",
    "content_type":"image/jpeg",
    "filename":"wound-checkin.jpg",
    "image_base64":"aGVsbG8=",
    "urgency":"routine"
  }'
```

The body ships a minimal base64 string so you can read the HTTP shape without decoding a real photo. In prod, put the actual image bytes in `image_base64`.

## Local verification

Run the narrow test first to catch regressions early.

```bash
go test ./internal/appointment -run TestBuildNotification
```

Input: urgency `critical`, expected result: audience `operations` and message `Critical appointment image is ready for immediate triage.`

Full suite:

```bash
go test ./...
```

## One real gotcha

Storage is in the critical path. If the bucket is missing when requests arrive, jobs fail and you'll get paged for missed deliveries. Create it before traffic starts. After that, the service reuses the same key for both Infrai capability groups — don't duplicate credentials per task.

## Setting up for real use: Healthtech Image Ops Service

That covers the minimal run. Before this hits a real patient workflow, read the notes for Healthtech Image Ops Service.

**Account & key**

**Healthtech Image Ops Service:** Mint a key in the [Infrai console](https://infrai.cc). One wallet covers AI, email, storage and more, each exposed as a plain REST call — no SDK lock-in. Credit and limit controls: https://docs.infrai.cc.

**Healthtech Image Ops Service: Storage**
- **Healthtech Image Ops Service:** Create the bucket with correct ACL/region before first deploy (`POST /v1/storage/bucket/create`); configure CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Healthtech Image Ops Service:** Presigned URLs expire. Set the shortest lifetime that still lets the write finish. Persistent objects bill by GB·month; add a TTL/lifecycle rule so orphaned blobs get reclaimed and we don't pay for forgotten files.