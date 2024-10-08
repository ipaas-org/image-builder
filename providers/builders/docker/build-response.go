package docker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ahmetb/go-cursor"
	orderedmap "github.com/wk8/go-ordered-map"
)

const (
	LayerMessagePrefix = "\u2023"
	errorPrefix        = "[ERROR]"
)

// BuildResponseBodyStreamAuxMessage contains the ImageBuild's aux data from buildResponse
type ResponseBodyStreamAuxMessage struct {
	// ID is response body stream aux's id
	ID string `json:"ID"`
}

// String return BuildResponseBodyStreamAuxMessage object as string
func (m *ResponseBodyStreamAuxMessage) String() string {
	if m.ID != "" {
		return fmt.Sprintf(" %s %s", LayerMessagePrefix, m.ID)
	}
	return ""
}

// BuildResponseBodyStreamAuxMessage contains the ImageBuild's aux data from buildResponse
type ResponseBodyStreamErrorDetailMessage struct {
	// ID is response body stream aux's id
	Message string `json:"message"`
}

// String return BuildResponseBodyStreamAuxMessage object as string
func (m *ResponseBodyStreamErrorDetailMessage) String() string {
	if m.Message != "" {
		return fmt.Sprintf("%s %s", errorPrefix, m.Message)
	}

	return ""
}

// ResponseBodyStreamMessage contains the ImageBuild's body data from buildResponse
type ResponseBodyStreamMessage struct {
	// Aux represents the aux value on response body stream message
	Aux *ResponseBodyStreamAuxMessage `json:"aux"`
	// ErrorDetail
	ErrorDetail *ResponseBodyStreamErrorDetailMessage `json:"errorDetail"`
	// ID identify layer
	ID string `json:"id"`
	// Progress contains the progress bar
	Progress string `json:"progress"`
	// Status represents the status value on response body stream message
	Status string `json:"status"`
	// Stream represents the stream value on response body stream message
	Stream string `json:"stream"`
}

// String return ResponseBodyStreamMessage object as string
func (m *ResponseBodyStreamMessage) String() string {
	if m.Status != "" {
		str := fmt.Sprintf("%s ", LayerMessagePrefix)
		if m.ID != "" {
			str = fmt.Sprintf("%s %s: ", str, strings.TrimSpace(m.ID))
		}
		str = fmt.Sprintf("%s %s ", str, strings.TrimSuffix(m.Status, "\n"))
		return str
	}
	if m.Stream != "" {
		return strings.TrimSpace(m.Stream)
	}
	if m.Aux != nil {
		return m.Aux.String()
	}
	if m.ErrorDetail != nil {
		return m.ErrorDetail.String()
	}

	return ""
}

// ProgressString returns progress bar
func (m *ResponseBodyStreamMessage) ProgressString() string {
	if m.Progress != "" {
		return strings.TrimSpace(m.Progress)
	}
	return ""
}

func ConvertOutput(imageOutput io.Reader) ([]byte, error) {
	writer := new(bytes.Buffer)
	scanner := bufio.NewScanner(imageOutput)
	lineBefore := ""
	lines := orderedmap.New()
	numLayers := 0

	for scanner.Scan() {
		streamMessage := &ResponseBodyStreamMessage{}
		line := scanner.Bytes()

		if err := json.Unmarshal(line, &streamMessage); err != nil {
			return nil, err
		}

		streamMessageStr := streamMessage.String()
		// fmt.Println("")
		if streamMessageStr != lineBefore && streamMessageStr != "" {
			if streamMessage.ID != "" {
				// override layer outputs on pull or push messages
				fmt.Fprintf(writer, "%s%s\n", cursor.MoveUp(numLayers+1), cursor.ClearEntireLine())

				lines.Set(streamMessage.ID, fmt.Sprint(streamMessage.String(), streamMessage.ProgressString()))
				for line := lines.Oldest(); line != nil; line = line.Next() {
					fmt.Fprintf(writer, "%s%s\n", line.Value, cursor.ClearLineRight())
				}
				numLayers = lines.Len()
			} else {
				fmt.Fprintf(writer, "%s%s\n", streamMessage.String(), streamMessage.ProgressString())
				lines = orderedmap.New()
				numLayers = 0
			}
		}

		lineBefore = streamMessageStr
	}

	return writer.Bytes(), nil
}
