package main

import (
	"encoding/xml"
	"io/ioutil"
	"os"
)

// Save is a convenience function to automatically serialize to default
// storage location.
func (rm *RouteMapping) Save() {
	rm.StoreToFile(rm.Storage)
}

// StoreToFile serializes the routemappings object to an XML file.
func (rm *RouteMapping) StoreToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		log.Error("Could not create file %s", err)
		return err
	}
	defer f.Close()

	output, err := xml.MarshalIndent(rm, "", "    ")
	if err != nil {
		log.Error("Error marshalling %s", err)
		return err
	}

	f.Write(output)
	return nil
}

func (rm *RouteMapping) restoreFromFile(path string) error {
	// If the file doesn't exist, just return.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Info("No file exists")
		rm.Routes = make([]Route, 0)
		return nil
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error("Error reading %s", err)
		return err
	}

	if err := xml.Unmarshal(data, &rm); err != nil {
		log.Error("Error unmarshalling %s", err)
		return err
	}

	return nil
}
