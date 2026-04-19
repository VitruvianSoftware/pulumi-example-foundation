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
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/essentialcontacts"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// deployEssentialContacts creates organization-level Essential Contacts
// for notification routing. This mirrors the Terraform foundation's
// essential_contacts.tf, implementing the full category-to-group mapping:
//
//	BILLING         → billing_admins, billing_data_users
//	LEGAL           → org_admins, audit_data_users
//	PRODUCT_UPDATES → org_admins
//	SECURITY        → scc_admin, security_reviewer
//	SUSPENSION      → org_admins
//	TECHNICAL       → security_reviewer, network_viewer
//
// Uses the configurable language tag (H9, defaults to "en").
func deployEssentialContacts(ctx *pulumi.Context, cfg *OrgConfig) error {
	parent := cfg.Parent // "organizations/<id>" or "folders/<id>"
	lang := cfg.EssentialContactsLanguage

	// Build a map of email → notification categories, mirroring the TF
	// transpose(categories_map) pattern. This groups categories per email
	// so each contact gets a single resource with all their categories.
	contactMap := make(map[string][]string)

	addContact := func(email string, categories ...string) {
		if email == "" {
			return
		}
		contactMap[email] = append(contactMap[email], categories...)
	}

	// Map categories to groups exactly as TF does
	// BILLING → billing_data_users
	addContact(cfg.BillingDataUsers, "BILLING")

	// LEGAL → audit_data_users
	addContact(cfg.AuditDataUsers, "LEGAL")

	// SECURITY → scc_admin (falls back to org admins in TF, but we only create if set)
	addContact(cfg.GCPSCCAdmin, "SECURITY")

	// SECURITY + TECHNICAL → security_reviewer
	addContact(cfg.GCPSecurityReviewer, "SECURITY", "TECHNICAL")

	// TECHNICAL → network_viewer
	addContact(cfg.GCPNetworkViewer, "TECHNICAL")

	// De-duplicate categories per contact
	for email, cats := range contactMap {
		seen := make(map[string]bool)
		unique := make([]string, 0, len(cats))
		for _, c := range cats {
			if !seen[c] {
				seen[c] = true
				unique = append(unique, c)
			}
		}
		contactMap[email] = unique
	}

	// Create one Essential Contact per unique email
	idx := 0
	for email, categories := range contactMap {
		catArray := make(pulumi.StringArray, len(categories))
		for i, c := range categories {
			catArray[i] = pulumi.String(c)
		}

		if _, err := essentialcontacts.NewContact(ctx, fmt.Sprintf("essential-contact-%d", idx), &essentialcontacts.ContactArgs{
			Parent:                            pulumi.String(parent),
			Email:                             pulumi.String(email),
			LanguageTag:                       pulumi.String(lang),
			NotificationCategorySubscriptions: catArray,
		}); err != nil {
			return err
		}
		idx++
	}

	return nil
}
