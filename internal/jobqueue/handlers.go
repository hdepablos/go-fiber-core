package jobqueue

import (
	"fmt"
)

type JobHandler func(data []byte) error

var handlers = make(map[string]JobHandler)

func RegisterHandler(jobType string, handler JobHandler) {
	handlers[jobType] = handler
}

func HandleJob(job JobMessage) error {
	handler, ok := handlers[job.Type]
	if !ok {
		return fmt.Errorf("handler no registrado para job type: %s", job.Type)
	}
	return handler(job.Data)
}
