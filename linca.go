package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

// Represents inotify event returned from inotifywait.
type notifyEvent struct {
	watched string
	events  []string
	file    string
}

// Make new notifyEvent.
func newEvent(watched string, events []string, file string) *notifyEvent {
	return &notifyEvent{watched, events, file}
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
	cmd := exec.Command("inotifywait", "-m", "--format", "%w\n%e\n%f", watchdir)
	output, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error starting inotifywait: %s", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Error starting inotifywait: %s", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go logStderr(stderr, &wg)

	scanner := bufio.NewScanner(output)
	cmd.Start()
	for scanner.Scan() {
		watched := scanner.Text()
		if !scanner.Scan() {
			log.Fatal("Incomplete output from inotifywait")
		}
		events := scanner.Text()
		if !scanner.Scan() {
			log.Fatal("Incomplete output from inotifywait")
		}
		file := scanner.Text()
		event := newEvent(watched, strings.Split(events, ","), file)
		out <- event
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error scanning inotifywait output: %s", err)
	}

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		log.Fatalf("inotifywait exited with non-zero: %s", err)
	}
}

// Log StderrPipe output.
func logStderr(stderr io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		log.Print(scanner.Text())
	}
}

// Handle received events and do any necessary action.
func linker(destdir string, events <-chan *notifyEvent) {
	for event := range events {
		log.Printf("Got event %#v", event)
		if event.file == "" || event.hasEvent("DELETE") {
			// This is an event we don't care about
			continue
		}
		file := path.Join(event.watched, event.file)
		dest := path.Join(destdir, event.file)
		fi, err := os.Stat(file)
		if err != nil {
			log.Printf("Stat error on %s: %s", file, err)
			continue
		}
		if event.hasEvent("CREATE") || event.hasEvent("MOVED_TO") {
			if fi.IsDir() {
				cmd := exec.Command("cp", "-al", file, destdir)
				cmd.Run()
			} else {
				err := os.Link(file, dest)
				if err != nil {
					log.Printf("Error linking file: %s", err)
				}
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
