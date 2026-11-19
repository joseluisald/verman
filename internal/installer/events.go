package installer

import (
	"fmt"
	"io"
)

type EventKind string

const (
	EventWarning          EventKind = "warning"
	EventDownloading      EventKind = "downloading"
	EventInstalled        EventKind = "installed"
	EventAlreadyInstalled EventKind = "already_installed"
	EventProgress         EventKind = "download_progress"
	EventExtracting       EventKind = "extracting"
	EventValidating       EventKind = "validating"
)

// Event é o contrato do núcleo para CLI/desktop. Não contém código de shell.
type Event struct {
	Kind      EventKind `json:"kind"`
	Language  string    `json:"language"`
	Version   string    `json:"version"`
	Directory string    `json:"directory,omitempty"`
	Message   string    `json:"message,omitempty"`
	Bytes     int64     `json:"bytes,omitempty"`
	Total     int64     `json:"total,omitempty"`
}

type EventSink func(Event) error

func emit(sink EventSink, event Event) error {
	if sink == nil {
		return nil
	}
	return sink(event)
}

// Compatibilidade com os consumidores antigos que recebem texto em um writer.
// O serviço de aplicação usa eventos diretamente, sem interpretar esta saída.
func legacyEvents(out io.Writer) EventSink {
	return func(event Event) error {
		var err error
		switch event.Kind {
		case EventWarning:
			_, err = fmt.Fprintln(out, "Aviso:", event.Message)
		case EventDownloading:
			_, err = fmt.Fprintf(out, "Baixando %s %s...\n", event.Language, event.Version)
		case EventInstalled:
			_, err = fmt.Fprintf(out, "✔ %s %s instalado em %s\n", event.Language, event.Version, event.Directory)
		case EventAlreadyInstalled:
			_, err = fmt.Fprintf(out, "✔ %s %s já está instalado.\n", event.Language, event.Version)
		}
		return err
	}
}
