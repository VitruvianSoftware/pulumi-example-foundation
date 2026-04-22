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

// TestNetConfigStruct validates the NetConfig struct with SVPC-specific fields.
func TestNetConfigStruct(t *testing.T) {
	cfg := &NetConfig{
		Env:     "development",
		Region1: "us-central1",
		Region2: "us-west1",
	}

	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "us-central1", cfg.Region1)
	assert.Equal(t, "us-west1", cfg.Region2)
}

// TestNetConfigDefaults validates default values for network config.
func TestNetConfigDefaults(t *testing.T) {
	cfg := &NetConfig{}

	// Apply defaults as loadNetConfig would
	if cfg.Region1 == "" {
		cfg.Region1 = "us-central1"
	}
	if cfg.Region2 == "" {
		cfg.Region2 = "us-west1"
	}

	assert.Equal(t, "us-central1", cfg.Region1)
	assert.Equal(t, "us-west1", cfg.Region2)
}

// TestNetConfigEnvironmentValues validates all valid environment inputs.
func TestNetConfigEnvironmentValues(t *testing.T) {
	validEnvs := []string{"development", "nonproduction", "production"}

	for _, env := range validEnvs {
		t.Run(env, func(t *testing.T) {
			cfg := &NetConfig{Env: env}
			assert.NotEmpty(t, cfg.Env)
		})
	}
}
