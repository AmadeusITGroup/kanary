package validation

//Result returns result of a Validation
type Result struct {
	IsFailed        bool
	ForceSuccessNow bool
	Comment         string
}
