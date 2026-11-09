package constants

type SurveyType int

const (
	SURVEY_MULTI_CHOICE SurveyType = iota
	SURVEY_FREE_TEXT
	SURVEY_TRUE_FALSE
	SURVEY_RANGE
)

func (i SurveyType) String() string {
	switch i {
	case SURVEY_MULTI_CHOICE:
		return "multi_choice"
	case SURVEY_FREE_TEXT:
		return "free_text"
	case SURVEY_TRUE_FALSE:
		return "true_false"
	case SURVEY_RANGE:
		return "range"
	}

	return "invalid_option"
}
