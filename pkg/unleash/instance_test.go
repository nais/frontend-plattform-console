package unleash

import (
	"testing"
	"time"

	unleashv1 "github.com/nais/unleasherator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	now         = time.Now()
	oneHourAgo  = now.Add(-1 * time.Hour)
	oneDayAgo   = now.Add(-24 * time.Hour)
	oneWeekAgo  = now.Add(-7 * 24 * time.Hour)
	oneMonthAgo = now.Add(-30 * 24 * time.Hour)
	oneYearAgo  = now.Add(-365 * 24 * time.Hour)

	tests = []struct {
		name string
		time metav1.Time
		want string
	}{
		{
			name: "less than a day",
			time: metav1.NewTime(oneHourAgo),
			want: "less than a day",
		},
		{
			name: "1 day",
			time: metav1.NewTime(oneDayAgo.Add(-1 * time.Hour)),
			want: "1 day",
		},
		{
			name: "2 days",
			time: metav1.NewTime(oneDayAgo.Add(-1 * 24 * time.Hour)),
			want: "2 days",
		},
		{
			name: "1 week",
			time: metav1.NewTime(oneWeekAgo),
			want: "1 week",
		},
		{
			name: "2 weeks",
			time: metav1.NewTime(oneWeekAgo.Add(-7 * 24 * time.Hour)),
			want: "2 weeks",
		},
		{
			name: "1 month",
			time: metav1.NewTime(oneMonthAgo),
			want: "1 month",
		},
		{
			name: "2 months",
			time: metav1.NewTime(oneMonthAgo.Add(-30 * 24 * time.Hour)),
			want: "2 months",
		},
		{
			name: "1 year",
			time: metav1.NewTime(oneYearAgo),
			want: "1 year",
		},
		{
			name: "2 years",
			time: metav1.NewTime(oneYearAgo.Add(-365 * 24 * time.Hour)),
			want: "2 years",
		},
	}
)

func TestHumanReadableAge(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := humanReadableAge(tt.time)
			if got != tt.want {
				t.Errorf("humanReadableAge(%v) = %v, want %v", tt.time, got, tt.want)
			}
		})
	}
}

func TestNewUnleashInstance(t *testing.T) {
	serverInstance := &unleashv1.Unleash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "test-namespace",
		},
	}
	instance := NewUnleashInstance(serverInstance)
	if instance.Name != "test-instance" {
		t.Errorf("Expected instance name to be 'test-instance', but got '%s'", instance.Name)
	}
	if instance.KubernetesNamespace != "test-namespace" {
		t.Errorf("Expected instance namespace to be 'test-namespace', but got '%s'", instance.KubernetesNamespace)
	}
}

func TestUnleashInstance_Age(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := UnleashInstance{
				CreatedAt: tt.time,
			}
			if got := u.Age(); got != tt.want {
				t.Errorf("UnleashInstance{CreatedAt: %v}.Age() = %v, want %v", tt.time, got, tt.want)
			}
		})
	}
}

func TestUnleashInstance_WebUrl(t *testing.T) {
	instance := &UnleashInstance{
		ServerInstance: &unleashv1.Unleash{
			Spec: unleashv1.UnleashSpec{
				WebIngress: unleashv1.UnleashIngressConfig{
					Enabled: true,
					Host:    "test.example.com",
				},
			},
		},
	}
	if got := instance.WebUrl(); got != "https://test.example.com/" {
		t.Errorf("UnleashInstance.WebUrl() = %v, want %v", got, "https://test.example.com/")
	}
}

func TestUnleashInstance_ApiUrl(t *testing.T) {
	instance := &UnleashInstance{
		ServerInstance: &unleashv1.Unleash{
			Spec: unleashv1.UnleashSpec{
				ApiIngress: unleashv1.UnleashIngressConfig{
					Enabled: true,
					Host:    "test.example.com",
				},
			},
		},
	}
	if got := instance.ApiUrl(); got != "https://test.example.com/api/" {
		t.Errorf("UnleashInstance.ApiUrl() = %v, want %v", got, "https://test.example.com/api/")
	}
}

func TestUnleashInstance_IsReady(t *testing.T) {
	instance := &UnleashInstance{
		ServerInstance: &unleashv1.Unleash{
			Status: unleashv1.UnleashStatus{
				Conditions: []metav1.Condition{
					{
						Type:   unleashv1.UnleashStatusConditionTypeReconciled,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   unleashv1.UnleashStatusConditionTypeConnected,
						Status: metav1.ConditionTrue,
					},
				},
			},
		},
	}
	if !instance.IsReady() {
		t.Errorf("UnleashInstance.IsReady() = %v, want %v", false, true)
	}
}

func TestUnleashInstance_Status(t *testing.T) {
	instance := &UnleashInstance{
		ServerInstance: &unleashv1.Unleash{
			Status: unleashv1.UnleashStatus{
				Conditions: []metav1.Condition{
					{
						Type:   unleashv1.UnleashStatusConditionTypeReconciled,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   unleashv1.UnleashStatusConditionTypeConnected,
						Status: metav1.ConditionTrue,
					},
				},
			},
		},
	}
	if got := instance.Status(); got != "Ready" {
		t.Errorf("UnleashInstance.Status() = %v, want %v", got, "Ready")
	}

	instance.ServerInstance.Status.Conditions[0].Status = metav1.ConditionFalse
	if got := instance.Status(); got != "Not ready" {
		t.Errorf("UnleashInstance.Status() = %v, want %v", got, "Not ready")
	}

	instance.ServerInstance = nil
	if got := instance.Status(); got != "Status unknown" {
		t.Errorf("UnleashInstance.Status() = %v, want %v", got, "Status unknown")
	}
}

func TestUnleashInstance_StatusLabel(t *testing.T) {
	instance := &UnleashInstance{
		ServerInstance: &unleashv1.Unleash{
			Status: unleashv1.UnleashStatus{
				Conditions: []metav1.Condition{
					{
						Type:   unleashv1.UnleashStatusConditionTypeReconciled,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   unleashv1.UnleashStatusConditionTypeConnected,
						Status: metav1.ConditionTrue,
					},
				},
			},
		},
	}
	if got := instance.StatusLabel(); got != "green" {
		t.Errorf("UnleashInstance.StatusLabel() = %v, want %v", got, "green")
	}

	instance.ServerInstance.Status.Conditions[0].Status = metav1.ConditionFalse
	if got := instance.StatusLabel(); got != "red" {
		t.Errorf("UnleashInstance.StatusLabel() = %v, want %v", got, "red")
	}

	instance.ServerInstance = nil
	if got := instance.StatusLabel(); got != "orange" {
		t.Errorf("UnleashInstance.StatusLabel() = %v, want %v", got, "orange")
	}
}
