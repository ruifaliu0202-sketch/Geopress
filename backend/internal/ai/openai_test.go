package ai

import "testing"

func TestResponsesAPIResponseOutputText(t *testing.T) {
	response := responsesAPIResponse{
		Output: []responsesAPIOutput{
			{Type: "reasoning"},
			{
				Type: "message",
				Content: []responsesAPIOutputContent{
					{Type: "output_text", Text: `{"title":"A"}`},
				},
			},
		},
	}

	if got := response.outputText(); got != `{"title":"A"}` {
		t.Fatalf("outputText() = %q", got)
	}
}

func TestResponsesAPIResponseOutputTextPrefersHelper(t *testing.T) {
	response := responsesAPIResponse{
		OutputText: `{"title":"B"}`,
		Output: []responsesAPIOutput{
			{
				Type: "message",
				Content: []responsesAPIOutputContent{
					{Type: "output_text", Text: `{"title":"A"}`},
				},
			},
		},
	}

	if got := response.outputText(); got != `{"title":"B"}` {
		t.Fatalf("outputText() = %q", got)
	}
}
