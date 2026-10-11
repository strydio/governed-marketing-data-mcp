package runs

const (
	CauseConversionTrackingUnderReporting = "conversion_tracking_under_reporting"
	CauseSafariTrackingFailure            = "safari_tracking_failure"
	CauseMissingConversionEvents          = "missing_conversion_events"
	CauseLowerQualityTraffic              = "lower_quality_traffic"
	CauseBudgetOrDeliveryChange           = "budget_or_delivery_change"
	CauseStaleOrIncompleteData            = "stale_or_incomplete_data"
	CauseUnknown                          = "unknown"
)

func AllowedCauses() []string {
	return []string{
		CauseConversionTrackingUnderReporting,
		CauseSafariTrackingFailure,
		CauseMissingConversionEvents,
		CauseLowerQualityTraffic,
		CauseBudgetOrDeliveryChange,
		CauseStaleOrIncompleteData,
		CauseUnknown,
	}
}

func IsAllowedCause(cause string) bool {
	for _, allowed := range AllowedCauses() {
		if cause == allowed {
			return true
		}
	}
	return false
}
