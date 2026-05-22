package types

const (
	TaskTypeBrowserSnapshot = "browser_snapshot"

	SignalCategoryPricing    = "pricing"
	SignalCategoryProtocol   = "protocol"
	SignalCategoryRelease    = "release"
	SignalCategoryCapability = "capability"
	SignalCategoryEcosystem  = "ecosystem"

	SeverityLow    = "low"
	SeverityMedium = "medium"
	SeverityHigh   = "high"

	SignalTypePricing = "pricing_change"
	SignalTypeFeature = "feature_change"
	SignalTypeRelease = "release_change"
	SignalTypeDoc     = "doc_change"
	SignalTypeHiring  = "hiring_change"
	SignalTypeOther   = "other"
)

// ValidSignalCategories lists allowed task/signal categories.
var ValidSignalCategories = []string{
	SignalCategoryPricing,
	SignalCategoryProtocol,
	SignalCategoryRelease,
	SignalCategoryCapability,
	SignalCategoryEcosystem,
}
