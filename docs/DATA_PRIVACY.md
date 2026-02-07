# Data Privacy Framework

This document describes the data privacy practices for the Ride Hailing Platform, including data classification, handling procedures, retention policies, and user rights.

---

## Data Classification

All data processed by the platform is classified into four categories based on sensitivity and regulatory requirements.

### Classification Levels

| Level | Description | Examples | Handling Requirements |
|-------|-------------|----------|----------------------|
| **Public** | Data intended for public access | App store listing, marketing materials, public APIs | No restrictions |
| **Internal** | Business data not for public disclosure | Internal metrics, aggregated analytics, system configs | Access logging, need-to-know basis |
| **Confidential (PII)** | Personally Identifiable Information | Name, email, phone, location history, ride history | Encryption, access controls, audit logging |
| **Restricted** | Highly sensitive financial/security data | Payment tokens, API keys, passwords, financial records | Maximum protection, strict access controls |

### Classification by Service

| Service | Data Types | Classification |
|---------|-----------|----------------|
| **Auth Service** | User credentials, JWT tokens, sessions | Restricted |
| **Rides Service** | Trip data, pickup/dropoff locations, fare details | Confidential |
| **Geo Service** | Real-time driver locations, geospatial data | Confidential |
| **Payments Service** | Stripe tokens, transaction records, wallet balances | Restricted |
| **Notifications Service** | Device tokens, notification preferences | Confidential |
| **Analytics Service** | Aggregated metrics, business reports | Internal |
| **Fraud Service** | Risk scores, fraud indicators | Restricted |

---

## PII Handling

### What PII We Collect

We collect the following categories of personal information:

| Category | Data Elements | Purpose |
|----------|--------------|---------|
| **Identity** | Full name, profile photo | User identification, driver-passenger matching |
| **Contact** | Phone number, email address | Account verification, communications, support |
| **Location** | GPS coordinates, pickup/dropoff addresses | Ride matching, navigation, ETA calculation |
| **Financial** | Payment method tokens, transaction history | Payment processing, refunds, receipts |
| **Device** | Device ID, push notification tokens, app version | Notifications, troubleshooting, security |
| **Usage** | Ride history, ratings, preferences | Service delivery, personalization, quality |

### Why We Collect PII

Each data element is collected for specific, documented purposes:

1. **Service Delivery**
   - Matching riders with nearby drivers
   - Calculating routes and ETAs
   - Processing payments and issuing receipts

2. **Safety and Security**
   - Verifying user identity
   - Fraud detection and prevention
   - Emergency assistance (sharing location with authorities if needed)

3. **Legal Compliance**
   - Tax reporting and invoicing
   - Regulatory record-keeping
   - Law enforcement cooperation (with valid legal process)

4. **Service Improvement**
   - Analyzing service quality
   - Improving matching algorithms
   - Optimizing pricing models

### How We Protect PII

Technical and organizational measures include:

| Measure | Implementation |
|---------|---------------|
| **Encryption at Rest** | AES-256 encryption for database storage |
| **Encryption in Transit** | TLS 1.3 for all API communications |
| **Access Controls** | Role-based access control (RBAC) |
| **Audit Logging** | All PII access logged with user, timestamp, action |
| **Pseudonymization** | Internal IDs instead of natural identifiers where possible |
| **Data Minimization** | Collect only what is necessary for stated purposes |
| **Network Segmentation** | Service mesh isolation via Istio |

---

## Data Retention Policies

### Retention Schedule

| Data Category | Active Period | Post-Deletion | Legal Basis |
|--------------|---------------|---------------|-------------|
| **User Profile** | While account active | 30 days after deletion request | Service delivery |
| **Ride History** | 7 years | Anonymized after deletion | Tax/legal requirements |
| **Location Data** | 90 days (detailed), 7 years (trip endpoints) | Deleted with account | Service delivery, legal |
| **Payment Records** | 7 years | Retained per financial regulations | Tax/legal requirements |
| **Support Tickets** | 3 years after resolution | Anonymized | Service improvement |
| **Audit Logs** | 7 years | N/A | Security, compliance |
| **Marketing Preferences** | While account active | Immediately deleted | Consent |

### Active User Data

For active users:

- **Real-time location**: Retained only during active session
- **Trip data**: Retained for service history and support
- **Payment methods**: Stripe tokens retained for convenience
- **Preferences**: Retained for personalization

### Inactive User Data

For users inactive for 24+ months:

- **Notification**: Email notification before any data action
- **Reduced Retention**: Non-essential data may be archived or deleted
- **Reactivation**: Full data restoration upon account reactivation
- **Grace Period**: 30-day grace period before archival actions

### Deleted Account Data

Upon account deletion request:

1. **Immediate** (within 72 hours):
   - Profile data removed from active systems
   - Payment methods deleted from Stripe
   - Push notification tokens invalidated

2. **Within 30 days**:
   - PII removed from primary databases
   - Backup propagation completed
   - Confirmation email sent

3. **Retained per legal requirements**:
   - Anonymized ride records (7 years for tax)
   - Financial transaction records (7 years)
   - Fraud investigation records (as required)

---

## Data Access Controls

### Role-Based Access Control (RBAC)

Access to data is controlled through role-based permissions:

| Role | Access Level | Data Scope |
|------|-------------|------------|
| **User** | Self-service | Own profile, ride history, payment methods |
| **Driver** | Limited | Own profile, assigned ride details, earnings |
| **Support Agent** | Read-only | User profiles, ride details (no payment data) |
| **Support Supervisor** | Read-only | Extended access for escalations |
| **Finance Team** | Read-only | Payment records, financial reports |
| **Engineering** | Role-dependent | System data, logs (no production PII access) |
| **Admin** | Full | All data (audit logged) |

### Audit Logging

All data access is logged with:

- **Who**: User ID, role, IP address
- **What**: Data type, record ID, fields accessed
- **When**: Timestamp with timezone
- **Why**: Action type (view, edit, delete, export)
- **Result**: Success or failure, error details

Audit logs are:
- Immutable (append-only)
- Retained for 7 years
- Regularly reviewed for anomalies
- Available for compliance audits

### Principle of Least Privilege

Access control principles:

1. **Default Deny**: No access unless explicitly granted
2. **Minimum Necessary**: Access only to data required for job function
3. **Time-Limited**: Elevated access expires automatically
4. **Separation of Duties**: Critical operations require multiple approvers
5. **Regular Review**: Quarterly access reviews and recertification

---

## Third-Party Data Sharing

### Approved Data Recipients

| Third Party | Data Shared | Purpose | Safeguards |
|-------------|-------------|---------|------------|
| **Stripe** | Payment tokens, transaction details | Payment processing | PCI-DSS Level 1, DPA |
| **Map Provider** | Route coordinates | Navigation, ETA | Data minimization, no PII |
| **Firebase (FCM)** | Device tokens, notification content | Push notifications | Encryption, DPA |
| **Twilio** | Phone numbers, SMS content | SMS notifications | DPA, data retention limits |
| **Sentry** | Error context, device info | Error tracking | PII scrubbing, self-hosted option |

### Data Sharing Agreements

All third-party processors must:

- Sign Data Processing Agreements (DPAs)
- Demonstrate adequate security measures
- Limit data use to specified purposes
- Delete data upon contract termination
- Support data subject rights requests
- Notify us of data breaches promptly

### No Data Sales

**We do not sell personal information.**

- No data sold for monetary consideration
- No data shared for third-party advertising
- No data broker relationships
- No cross-context behavioral advertising

---

## User Rights and Requests

### How to Request Data Export

Users can export their data through:

1. **Self-Service (Recommended)**
   - Navigate to Settings > Privacy > Download My Data
   - Select data categories to export
   - Receive download link within 24 hours

2. **Support Request**
   - Email: privacy@ridehailing.example.com
   - Subject: "Data Export Request"
   - Include: Account email, verification information

3. **API Access**
   - Authenticated API endpoint for programmatic export
   - Returns JSON format with all user data

### How to Request Deletion

Account and data deletion options:

1. **Self-Service**
   - Navigate to Settings > Account > Delete Account
   - Confirm via email or SMS verification
   - 30-day grace period before permanent deletion

2. **Support Request**
   - Email: privacy@ridehailing.example.com
   - Subject: "Account Deletion Request"
   - Identity verification required

3. **Partial Deletion**
   - Delete specific data categories while keeping account
   - Available for ride history, payment methods, etc.

### Response Timeframes

| Request Type | Response Time | Completion Time |
|-------------|---------------|-----------------|
| **Data Access/Export** | Acknowledgment within 3 days | Complete within 30 days |
| **Data Correction** | Acknowledgment within 3 days | Complete within 7 days |
| **Data Deletion** | Acknowledgment within 3 days | Complete within 30 days |
| **Opt-Out Requests** | Acknowledgment within 3 days | Complete within 15 days |

### Verification Requirements

To protect against unauthorized requests:

- **Account Holders**: Login verification or email/SMS code
- **Authorized Agents**: Written authorization + identity verification
- **Bulk Requests**: Additional verification steps may apply

---

## Data Breach Response

### Breach Notification

In the event of a data breach:

| Audience | Notification Timeline | Method |
|----------|----------------------|--------|
| **Affected Users** | Within 72 hours | Email, in-app notification |
| **Regulatory Authorities** | Within 72 hours (GDPR) | Official channels |
| **Law Enforcement** | As required | Direct communication |

### Breach Response Process

1. **Detection and Containment** - Immediate isolation of affected systems
2. **Assessment** - Determine scope, data types, and affected users
3. **Notification** - Notify required parties per regulatory timelines
4. **Remediation** - Address root cause, enhance controls
5. **Post-Incident Review** - Document lessons learned, update procedures

---

## Privacy by Design

Our development practices incorporate privacy from the start:

- **Data Minimization** - Collect only necessary data
- **Purpose Limitation** - Use data only for stated purposes
- **Storage Limitation** - Delete data when no longer needed
- **Accuracy** - Maintain accurate and up-to-date information
- **Security by Default** - Encryption and access controls enabled by default
- **Transparency** - Clear privacy notices and consent mechanisms

---

## Contact Information

For privacy-related inquiries:

| Topic | Contact |
|-------|---------|
| **Privacy Requests** | privacy@ridehailing.example.com |
| **Data Protection Officer** | dpo@ridehailing.example.com |
| **Security Concerns** | security@ridehailing.example.com |
| **General Support** | support@ridehailing.example.com |

---

*Last Updated: February 2026*
*Document Version: 1.0*
*Next Review: August 2026*
