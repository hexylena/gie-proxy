package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
)

// Save is a convenience function to automatically serialize to default
// storage location.
func (rm *RouteMapping) Save() {
	// Already handled errors in StoreToFile()'s logging
	_ = rm.StoreToFile(rm.Storage)
}

// StoreToFile serializes the routemappings object to an XML file.
func (rm *RouteMapping) StoreToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		log.Error(fmt.Sprintf("Could not create file %s", err))
		return err
	}

	output, err := xml.MarshalIndent(rm, "", "    ")
	if err != nil {
		log.Error(fmt.Sprintf("Error marshalling %s", err))
		return err
	}

	_, err = f.Write(output)
	if err != nil {
		log.Error(fmt.Sprintf("Error writing %s", err))
		return err
	}

	err = f.Close()
	if err != nil {
		log.Error(fmt.Sprintf("Error closing %s", err))
		return err
	}
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
		log.Error(fmt.Sprintf("Error reading %s", err))
		return err
	}

	// Unmarshal into a separate object, because we only want the routes
	rm2 := &RouteMapping{}
	if err := xml.Unmarshal(data, &rm2); err != nil {
		log.Error(fmt.Sprintf("Error unmarshalling %s", err))
		return err
	}

	rm.Routes = rm2.Routes

	return nil
}
