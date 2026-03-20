package watcher

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

func StartFolderWatcher(path string, fileQueue chan<- string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		defer watcher.Close()

		log.Printf("Starting folder watcher on: %s\n", path)

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op.Has(fsnotify.Create) {
					log.Println("Created file:", event.Name)

					fileQueue <- event.Name
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher error:", err)
			}
		}
	}()

	return watcher.Add(path)
}
