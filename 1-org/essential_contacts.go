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
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/essentialcontacts"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// deployEssentialContacts creates organization-level Essential Contacts
// for notification routing. This mirrors the Terraform foundation's
// essential_contacts.tf, ensuring billing, security, and legal notifications
// reach the appropriate governance groups.
func deployEssentialContacts(ctx *pulumi.Context, cfg *OrgConfig) error {
	parent := cfg.Parent // "organizations/<id>" or "folders/<id>"

	// Billing notifications → billing_data_users group
	if cfg.BillingDataUsers != "" {
		if _, err := essentialcontacts.NewContact(ctx, "essential-contact-billing", &essentialcontacts.ContactArgs{
			Parent:                     pulumi.String(parent),
			Email:                      pulumi.String(cfg.BillingDataUsers),
			LanguageTag:                pulumi.String("en"),
			NotificationCategorySubscriptions: pulumi.StringArray{
				pulumi.String("BILLING"),
			},
		}); err != nil {
			return err
		}
	}

	// Security notifications → security_reviewer group
	if cfg.GCPSecurityReviewer != "" {
		if _, err := essentialcontacts.NewContact(ctx, "essential-contact-security", &essentialcontacts.ContactArgs{
			Parent:                     pulumi.String(parent),
			Email:                      pulumi.String(cfg.GCPSecurityReviewer),
			LanguageTag:                pulumi.String("en"),
			NotificationCategorySubscriptions: pulumi.StringArray{
				pulumi.String("SECURITY"),
				pulumi.String("TECHNICAL"),
			},
		}); err != nil {
			return err
		}
	}

	return nil
}
