package abis

type AdvanceError struct {
	FieldName string
	Err       error
}

func (e AdvanceError) Error() string {
	if e.Err == nil {
		return "advance: " + e.FieldName
	} else {
		return "advance: " + e.FieldName + ": " + e.Err.Error()
	}
}

func NewAdvanceError(fieldName string, err error) error {
	return &AdvanceError{fieldName, err}
}
