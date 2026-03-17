package workflow

import "errors"

var (
	ErrMissingWorkflowFile        = errors.New("missing_workflow_file")
	ErrWorkflowParseError         = errors.New("workflow_parse_error")
	ErrWorkflowFrontMatterNotAMap = errors.New("workflow_front_matter_not_a_map")
)
