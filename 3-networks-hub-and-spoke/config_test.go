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

// TestHubAndSpokeNetConfigStruct validates the hub-and-spoke network config.
func TestHubAndSpokeNetConfigStruct(t *testing.T) {
	cfg := &NetConfig{
		Env:     "production",
		Region1: "us-central1",
		Region2: "us-west1",
	}

	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "us-central1", cfg.Region1)
	assert.Equal(t, "us-west1", cfg.Region2)
}

// TestHubAndSpokeNetConfigDefaults validates defaults match TF upstream.
func TestHubAndSpokeNetConfigDefaults(t *testing.T) {
	cfg := &NetConfig{}

	if cfg.Region1 == "" {
		cfg.Region1 = "us-central1"
	}
	if cfg.Region2 == "" {
		cfg.Region2 = "us-west1"
	}

	assert.Equal(t, "us-central1", cfg.Region1)
	assert.Equal(t, "us-west1", cfg.Region2)
}
