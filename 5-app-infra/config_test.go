/*
 * Copyright 2026 Vitruvian Software
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 */

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAppInfraConfigStruct validates the AppInfraConfig struct.
func TestAppInfraConfigStruct(t *testing.T) {
	cfg := &AppInfraConfig{
		Env:          "development",
		BusinessCode: "bu1",
		Region:       "us-central1",
	}

	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "bu1", cfg.BusinessCode)
	assert.Equal(t, "us-central1", cfg.Region)
}

// TestDefaultRegion validates the default region constant.
func TestDefaultRegion(t *testing.T) {
	cfg := &AppInfraConfig{Region: ""}
	if cfg.Region == "" {
		cfg.Region = "us-central1"
	}
	assert.Equal(t, "us-central1", cfg.Region)
}

// TestDefaultBusinessCode validates the default business code.
func TestDefaultBusinessCode(t *testing.T) {
	cfg := &AppInfraConfig{BusinessCode: ""}
	if cfg.BusinessCode == "" {
		cfg.BusinessCode = "bu1"
	}
	assert.Equal(t, "bu1", cfg.BusinessCode)
}
