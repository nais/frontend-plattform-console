package utils

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HumanReadableAge(age metav1.Time) string {
	now := time.Now()
	diff := now.Sub(age.Time)

	if diff.Hours() < 24 {
		return "less than a day"
	} else if diff.Hours() < 24*7 {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	} else if diff.Hours() < 24*30 {
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week"
		}
		return fmt.Sprintf("%d weeks", weeks)
	} else if diff.Hours() < 24*365 {
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month"
		}
		return fmt.Sprintf("%d months", months)
	} else {
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year"
		}
		return fmt.Sprintf("%d years", years)
	}
}
