package domain

import (
	"fmt"
	"regexp"
	"time"
)

var campaignSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func ValidateCommonQuery(q CommonQuery, maxDateRangeDays int) error {
	if !campaignSlugPattern.MatchString(q.Campaign) {
		return fmt.Errorf("invalid campaign")
	}
	if q.Market != "UK" && q.Market != "US" {
		return fmt.Errorf("invalid market")
	}
	start, err := time.Parse(time.DateOnly, q.StartDate)
	if err != nil {
		return fmt.Errorf("invalid start date")
	}
	end, err := time.Parse(time.DateOnly, q.EndDate)
	if err != nil {
		return fmt.Errorf("invalid end date")
	}
	if end.Before(start) {
		return fmt.Errorf("invalid date range")
	}
	days := int(end.Sub(start).Hours()/24) + 1
	if days < 1 {
		return fmt.Errorf("invalid date range")
	}
	if maxDateRangeDays > 0 && days > maxDateRangeDays {
		return fmt.Errorf("date range exceeds maximum")
	}
	return nil
}
