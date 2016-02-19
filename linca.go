package main

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Represents inotify event returned from inotifywait.
type notifyEvent struct {
	events []string
	file   string
}

// Make new notifyEvent.
func newEvent(events []string, file string) *notifyEvent {
	return &notifyEvent{events, file}
}

// Check if notify event contains an event of the given type.
func (e *notifyEvent) hasEvent(event string) bool {
	for _, eventName := range e.events {
		if eventName == event {
			return true
		}
	}
	return false
}

// Mkdir, existing is okay.
func mkdirp(dir string) error {
	err := os.Mkdir(dir, 0777) // Use user umask.
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

// Watch directory with inotify and send events over channel.
func watcher(watchdir string, out chan<- *notifyEvent) {
	defer close(out)
	cmd := exec.Command("inotifywait", "-m", "--format", "%e\n%f", watchdir)
	output, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error trying to watch %s: %s", watchdir, err)
	}
	scanner := bufio.NewScanner(output)
	cmd.Start()
	for scanner.Scan() {
		events := scanner.Text()
		if !scanner.Scan() {
			log.Fatal("Incomplete output from inotifywait")
		}
		file := scanner.Text()
		event := newEvent(strings.Split(events, ","), file)
		out <- event
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error scanning inotifywait output: %s", err)
	}
}

// Handle received events and do any necessary action.
func linker(destdir string, events <-chan *notifyEvent) {
	for event := range events {
		fi, err := os.Stat(event.file)
		if err != nil {
			log.Printf("Stat error on %s", event.file)
			continue
		}
		if event.hasEvent("create") {
			if fi.IsDir() {
				err := mkdirp(destdir)
				if err != nil {
					log.Printf("Error creating dir: %s", err)
				}
			} else {
				err := os.Link(event.file, destdir)
				if err != nil {
					log.Printf("Error linking file: %s", err)
				}
			}
		}
		if event.hasEvent("modify") {
			if fi.IsDir() {
				cmd := exec.Command("cp", "-al", event.file, destdir)
				cmd.Run()
			} else {
				// File modified.  Already linked, so do nothing.
			}
		}
	}
}

func main() {
	watchdir := os.Args[1]
	destdir := os.Args[2]

	err := mkdirp(destdir)
	if err != nil {
		log.Fatalf("Error making destdir: %s", err)
	}

	events := make(chan *notifyEvent)
	go watcher(watchdir, events)
	linker(destdir, events)
}
