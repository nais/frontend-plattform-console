package unleash

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
			assert.Equal(t, tt.want, got)
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
	assert.Equal(t, "test-instance", instance.Name)
	assert.Equal(t, "test-namespace", instance.KubernetesNamespace)
}

func TestUnleashInstance_Age(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := UnleashInstance{
				CreatedAt: tt.time,
			}
			got := u.Age()
			assert.Equal(t, tt.want, got)
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
	got := instance.WebUrl()
	assert.Equal(t, "https://test.example.com/", got)
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
	got := instance.ApiUrl()
	assert.Equal(t, "https://test.example.com/api/", got)
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
	got := instance.IsReady()
	assert.True(t, got)
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
	got := instance.Status()
	assert.Equal(t, "Ready", got)

	instance.ServerInstance.Status.Conditions[0].Status = metav1.ConditionFalse
	got = instance.Status()
	assert.Equal(t, "Not ready", got)

	instance.ServerInstance = nil
	got = instance.Status()
	assert.Equal(t, "Status unknown", got)
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
	got := instance.StatusLabel()
	assert.Equal(t, "green", got)

	instance.ServerInstance.Status.Conditions[0].Status = metav1.ConditionFalse
	got = instance.StatusLabel()
	assert.Equal(t, "red", got)

	instance.ServerInstance = nil
	got = instance.StatusLabel()
	assert.Equal(t, "orange", got)
}
