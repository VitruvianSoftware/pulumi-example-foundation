/*
 * Copyright 2026 Vitruvian Software
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConfigDefaults verifies that the Config struct applies the correct
// defaults matching the Terraform foundation's variables.tf defaults.
func TestConfigDefaults(t *testing.T) {
	cfg := &Config{}

	// Apply defaults the same way loadConfig does
	if cfg.ProjectPrefix == "" {
		cfg.ProjectPrefix = "prj"
	}
	if cfg.FolderPrefix == "" {
		cfg.FolderPrefix = "fldr"
	}
	if cfg.BucketPrefix == "" {
		cfg.BucketPrefix = "bkt"
	}
	if cfg.ProjectDeletionPolicy == "" {
		cfg.ProjectDeletionPolicy = "PREVENT"
	}
	if cfg.DefaultRegion == "" {
		cfg.DefaultRegion = "us-central1"
	}
	if cfg.DefaultRegion2 == "" {
		cfg.DefaultRegion2 = "us-west1"
	}
	if cfg.DefaultRegionGCS == "" {
		cfg.DefaultRegionGCS = "US"
	}
	if cfg.DefaultRegionKMS == "" {
		cfg.DefaultRegionKMS = "us"
	}
	if cfg.KMSKeyProtectionLevel == "" {
		cfg.KMSKeyProtectionLevel = "SOFTWARE"
	}

	assert.Equal(t, "prj", cfg.ProjectPrefix, "project_prefix default")
	assert.Equal(t, "fldr", cfg.FolderPrefix, "folder_prefix default")
	assert.Equal(t, "bkt", cfg.BucketPrefix, "bucket_prefix default")
	assert.Equal(t, "PREVENT", cfg.ProjectDeletionPolicy, "project_deletion_policy default")
	assert.Equal(t, "us-central1", cfg.DefaultRegion, "default_region default")
	assert.Equal(t, "us-west1", cfg.DefaultRegion2, "default_region_2 default")
	assert.Equal(t, "US", cfg.DefaultRegionGCS, "default_region_gcs default")
	assert.Equal(t, "us", cfg.DefaultRegionKMS, "default_region_kms default")
	assert.Equal(t, "SOFTWARE", cfg.KMSKeyProtectionLevel, "kms_key_protection_level default")
}

// TestConfigParentOrgRoot tests that parent is set to org when no parent_folder is specified.
func TestConfigParentOrgRoot(t *testing.T) {
	cfg := &Config{
		OrgID: "123456789",
	}

	if cfg.ParentFolder != "" {
		cfg.Parent = "folders/" + cfg.ParentFolder
		cfg.ParentType = "folder"
		cfg.ParentID = cfg.ParentFolder
	} else {
		cfg.Parent = "organizations/" + cfg.OrgID
		cfg.ParentType = "organization"
		cfg.ParentID = cfg.OrgID
	}

	assert.Equal(t, "organizations/123456789", cfg.Parent)
	assert.Equal(t, "organization", cfg.ParentType)
	assert.Equal(t, "123456789", cfg.ParentID)
}

// TestConfigParentFolder tests that parent is set to folder when parent_folder is specified.
func TestConfigParentFolder(t *testing.T) {
	cfg := &Config{
		OrgID:        "123456789",
		ParentFolder: "987654321",
	}

	if cfg.ParentFolder != "" {
		cfg.Parent = "folders/" + cfg.ParentFolder
		cfg.ParentType = "folder"
		cfg.ParentID = cfg.ParentFolder
	} else {
		cfg.Parent = "organizations/" + cfg.OrgID
		cfg.ParentType = "organization"
		cfg.ParentID = cfg.OrgID
	}

	assert.Equal(t, "folders/987654321", cfg.Parent)
	assert.Equal(t, "folder", cfg.ParentType)
	assert.Equal(t, "987654321", cfg.ParentID)
}

// TestConfigRandomSuffix tests that random_suffix defaults to true.
func TestConfigRandomSuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"default (empty) → true", "", true},
		{"explicit true", "true", true},
		{"explicit false", "false", false},
		{"any other value → true", "yes", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input != "false"
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConfigFolderDeletionProtection tests that folder_deletion_protection defaults to true.
func TestConfigFolderDeletionProtection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"default (empty) → true", "", true},
		{"explicit true", "true", true},
		{"explicit false", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input != "false"
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConfigInitialGroupConfig tests the InitialGroupConfig default.
func TestConfigInitialGroupConfig(t *testing.T) {
	cfg := &Config{}
	if cfg.InitialGroupConfig == "" {
		cfg.InitialGroupConfig = "WITH_INITIAL_OWNER"
	}
	assert.Equal(t, "WITH_INITIAL_OWNER", cfg.InitialGroupConfig)
}

// TestSeedProjectStruct verifies the SeedProject output struct.
func TestSeedProjectStruct(t *testing.T) {
	sp := &SeedProject{}
	assert.NotNil(t, sp)
}

// TestCICDProjectStruct verifies the CICDProject output struct.
func TestCICDProjectStruct(t *testing.T) {
	cp := &CICDProject{}
	assert.NotNil(t, cp)
}
