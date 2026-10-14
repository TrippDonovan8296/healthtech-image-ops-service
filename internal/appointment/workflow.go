package appointment

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/example/healthtech-image-ops-service/internal/infrai"
)

type Optimizer interface {
	Image() infrai.ImageClient
	Storage() infrai.StorageClient
	PutPresigned(ctx context.Context, url string, raw []byte, contentType string) error
}

type Service struct {
	client Optimizer
	bucket string
}

type OptimizeRequest struct {
	AppointmentID string `json:"appointment_id"`
	PatientID     string `json:"patient_id"`
	ContentType   string `json:"content_type"`
	Filename      string `json:"filename"`
	ImageBase64   string `json:"image_base64"`
	Urgency       string `json:"urgency"`
}

type Notification struct {
	Audience string `json:"audience"`
	Message  string `json:"message"`
	Delivery string `json:"delivery"`
}

type OptimizeResult struct {
	AppointmentID     string       `json:"appointment_id"`
	Status            string       `json:"status"`
	OptimizedAssetKey string       `json:"optimized_asset_key"`
	Notification      Notification `json:"notification"`
}

func NewService(client Optimizer, bucket string) *Service {
	return &Service{client: client, bucket: bucket}
}

func (s *Service) OptimizeAppointmentImage(ctx context.Context, req OptimizeRequest) (OptimizeResult, error) {
	compressed, err := s.client.Image().Compress(ctx, req.ImageBase64)
	if err != nil {
		return OptimizeResult{}, err
	}

	raw, err := infrai.DecodeBase64Image(compressed.Image)
	if err != nil {
		return OptimizeResult{}, err
	}

	assetKey := buildOptimizedAssetKey(req.AppointmentID, req.Filename)
	presigned, err := s.client.Storage().PresignPut(ctx, s.bucket, assetKey, req.ContentType, req.AppointmentID)
	if err != nil {
		return OptimizeResult{}, err
	}

	if err := s.client.PutPresigned(ctx, presigned.URL, raw, req.ContentType); err != nil {
		return OptimizeResult{}, err
	}

	return OptimizeResult{
		AppointmentID:     req.AppointmentID,
		Status:            "ready",
		OptimizedAssetKey: assetKey,
		Notification:      BuildNotification(req.Urgency),
	}, nil
}

func BuildNotification(urgency string) Notification {
	switch strings.ToLower(strings.TrimSpace(urgency)) {
	case "critical":
		return Notification{
			Audience: "operations",
			Message:  "Critical appointment image is ready for immediate triage.",
			Delivery: "pager",
		}
	case "same_day":
		return Notification{
			Audience: "operations",
			Message:  "Same-day appointment image is ready for queue review.",
			Delivery: "dashboard",
		}
	default:
		return Notification{
			Audience: "patient",
			Message:  "Your appointment image is ready for review.",
			Delivery: "portal",
		}
	}
}

func buildOptimizedAssetKey(appointmentID, filename string) string {
	cleanName := path.Base(filename)
	if cleanName == "." || cleanName == "/" || cleanName == "" {
		cleanName = "image.bin"
	}
	return fmt.Sprintf("appointments/%s/optimized/%s", appointmentID, cleanName)
}
