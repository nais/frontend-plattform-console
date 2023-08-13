package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
			got := HumanReadableAge(tt.time)
			assert.Equal(t, tt.want, got)
		})
	}
}
