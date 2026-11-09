package constants

type QuestionType int

const (
	QUIZ_MULTI_CHOICE QuestionType = iota
	QUIZ_FREE_TEXT
	QUIZ_TRUE_FALSE
	QUIZ_RANGE
)

func (i QuestionType) String() string {
	switch i {
	case QUIZ_MULTI_CHOICE:
		return "multi_choice"
	case QUIZ_FREE_TEXT:
		return "free_text"
	case QUIZ_TRUE_FALSE:
		return "true_false"
	case QUIZ_RANGE:
		return "range"
	}

	return "invalid_option"
}
