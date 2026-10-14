package appointment

import "testing"

func TestBuildNotification(t *testing.T) {
	tests := []struct {
		name     string
		urgency  string
		expected Notification
	}{
		{
			name:    "critical routes to operations pager",
			urgency: "critical",
			expected: Notification{
				Audience: "operations",
				Message:  "Critical appointment image is ready for immediate triage.",
				Delivery: "pager",
			},
		},
		{
			name:    "same day stays operational but less urgent",
			urgency: "same_day",
			expected: Notification{
				Audience: "operations",
				Message:  "Same-day appointment image is ready for queue review.",
				Delivery: "dashboard",
			},
		},
		{
			name:    "routine is patient safe",
			urgency: "routine",
			expected: Notification{
				Audience: "patient",
				Message:  "Your appointment image is ready for review.",
				Delivery: "portal",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildNotification(tc.urgency)
			if got != tc.expected {
				t.Fatalf("BuildNotification(%q) = %+v, want %+v", tc.urgency, got, tc.expected)
			}
		})
	}
}

func TestBuildOptimizedAssetKey(t *testing.T) {
	got := buildOptimizedAssetKey("apt_2048", "wound-checkin.jpg")
	want := "appointments/apt_2048/optimized/wound-checkin.jpg"
	if got != want {
		t.Fatalf("buildOptimizedAssetKey() = %q, want %q", got, want)
	}
}
