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
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/pubsub"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/securitycenter"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// deploySCCNotification creates the Security Command Center notification
// pipeline: a Pub/Sub topic + subscription, and an SCC notification config
// that streams all active findings to the topic.
// This mirrors the Terraform foundation's scc_notification.tf.
func deploySCCNotification(ctx *pulumi.Context, cfg *OrgConfig, sccProjectID pulumi.StringOutput) error {
	// 1. Pub/Sub Topic for SCC findings
	sccTopic, err := pubsub.NewTopic(ctx, "scc-notification-topic", &pubsub.TopicArgs{
		Project: sccProjectID,
		Name:    pulumi.String("top-scc-notification"),
	})
	if err != nil {
		return err
	}

	// 2. Pub/Sub Subscription for consuming SCC findings
	if _, err := pubsub.NewSubscription(ctx, "scc-notification-subscription", &pubsub.SubscriptionArgs{
		Project: sccProjectID,
		Name:    pulumi.String("sub-scc-notification"),
		Topic:   sccTopic.Name,
	}); err != nil {
		return err
	}

	// 3. SCC V2 Notification Config — streams findings to Pub/Sub
	if _, err := securitycenter.NewV2OrganizationNotificationConfig(ctx, "scc-notification", &securitycenter.V2OrganizationNotificationConfigArgs{
		Organization: pulumi.String(cfg.OrgID),
		ConfigId:     pulumi.String("scc-notify-all"),
		Description:  pulumi.String("SCC Notification for all active findings"),
		PubsubTopic:  sccTopic.ID(),
		StreamingConfig: &securitycenter.V2OrganizationNotificationConfigStreamingConfigArgs{
			Filter: pulumi.String(cfg.SCCNotificationFilter),
		},
	}); err != nil {
		return err
	}

	return nil
}
