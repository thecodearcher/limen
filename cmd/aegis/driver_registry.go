package main

import (
	"fmt"
)

type driverRegistry struct {
	drivers map[string]Driver
}

var (
	driversRegistry *driverRegistry
)

func initDriverRegistry() {
	driversRegistry = &driverRegistry{
		drivers: make(map[string]Driver),
	}

	driversRegistry.registerDefaults()
}

func (r *driverRegistry) registerDriver(driver Driver) {
	r.drivers[driver.Name()] = driver
}

func (r *driverRegistry) getDriver(name string) (Driver, error) {
	driver, ok := r.drivers[name]
	if !ok {
		return nil, fmt.Errorf("driver not found: %s", name)
	}

	return driver, nil
}

func (r *driverRegistry) registerDefaults() {
	r.registerDriver(NewPostgresDriver())
	r.registerDriver(NewMySQLDriver())
}

func GetDriversRegistry() *driverRegistry {
	if driversRegistry == nil {
		initDriverRegistry()
	}
	return driversRegistry
}
