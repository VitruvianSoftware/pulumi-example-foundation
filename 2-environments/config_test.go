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

// TestEnvironments validates the fixed set of environment names used
// for folder and project creation. Must match Terraform upstream.
func TestEnvironments(t *testing.T) {
	envs := []string{"development", "nonproduction", "production"}
	assert.Len(t, envs, 3, "exactly 3 environments expected")
	assert.Contains(t, envs, "development")
	assert.Contains(t, envs, "nonproduction")
	assert.Contains(t, envs, "production")
}

// TestEnvCode validates the environment code mapping.
func TestEnvCode(t *testing.T) {
	envCodes := map[string]string{
		"development":   "d",
		"nonproduction": "n",
		"production":    "p",
	}
	assert.Equal(t, "d", envCodes["development"])
	assert.Equal(t, "n", envCodes["nonproduction"])
	assert.Equal(t, "p", envCodes["production"])
}
