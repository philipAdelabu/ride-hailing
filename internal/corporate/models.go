package corporate

import (
	"time"

	"github.com/google/uuid"
)

// AccountStatus represents the status of a corporate account
type AccountStatus string

const (
	AccountStatusPending   AccountStatus = "pending"
	AccountStatusActive    AccountStatus = "active"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusClosed    AccountStatus = "closed"
)

// EmployeeRole represents the role of an employee in a corporate account
type EmployeeRole string

const (
	EmployeeRoleAdmin   EmployeeRole = "admin"
	EmployeeRoleManager EmployeeRole = "manager"
	EmployeeRoleUser    EmployeeRole = "user"
)

// BillingCycle represents billing cycle options
type BillingCycle string

const (
	BillingCycleWeekly   BillingCycle = "weekly"
	BillingCycleMonthly  BillingCycle = "monthly"
	BillingCycleQuarterly BillingCycle = "quarterly"
)

// PolicyType represents the type of ride policy
type PolicyType string

const (
	PolicyTypeTimeRestriction   PolicyType = "time_restriction"
	PolicyTypeLocationRestriction PolicyType = "location_restriction"
	PolicyTypeAmountLimit       PolicyType = "amount_limit"
	PolicyTypeRideTypeRestriction PolicyType = "ride_type_restriction"
	PolicyTypeApprovalRequired  PolicyType = "approval_required"
)

// CorporateAccount represents a company's account
type CorporateAccount struct {
	ID                uuid.UUID     `json:"id" db:"id"`
	Name              string        `json:"name" db:"name"`
	LegalName         string        `json:"legal_name" db:"legal_name"`
	TaxID             *string       `json:"tax_id,omitempty" db:"tax_id"`
	Status            AccountStatus `json:"status" db:"status"`

	// Contact Information
	PrimaryEmail      string        `json:"primary_email" db:"primary_email"`
	PrimaryPhone      *string       `json:"primary_phone,omitempty" db:"primary_phone"`
	BillingEmail      string        `json:"billing_email" db:"billing_email"`
	Address           *Address      `json:"address,omitempty"`

	// Billing Configuration
	BillingCycle      BillingCycle  `json:"billing_cycle" db:"billing_cycle"`
	PaymentTermDays   int           `json:"payment_term_days" db:"payment_term_days"` // Net 15, 30, 45, etc.
	CreditLimit       float64       `json:"credit_limit" db:"credit_limit"`
	CurrentBalance    float64       `json:"current_balance" db:"current_balance"`

	// Pricing
	DiscountPercent   float64       `json:"discount_percent" db:"discount_percent"` // Corporate discount
	CustomRates       bool          `json:"custom_rates" db:"custom_rates"`

	// Settings
	RequireApproval   bool          `json:"require_approval" db:"require_approval"`
	RequireCostCenter bool          `json:"require_cost_center" db:"require_cost_center"`
	RequireProjectCode bool         `json:"require_project_code" db:"require_project_code"`
	AllowPersonalRides bool         `json:"allow_personal_rides" db:"allow_personal_rides"`

	// Integration
	ExpenseSystemID   *string       `json:"expense_system_id,omitempty" db:"expense_system_id"` // Concur, SAP, etc.
	SSOEnabled        bool          `json:"sso_enabled" db:"sso_enabled"`
	SSOProvider       *string       `json:"sso_provider,omitempty" db:"sso_provider"`

	// Metadata
	LogoURL           *string       `json:"logo_url,omitempty" db:"logo_url"`
	Industry          *string       `json:"industry,omitempty" db:"industry"`
	CompanySize       *string       `json:"company_size,omitempty" db:"company_size"` // small, medium, large, enterprise

	CreatedAt         time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at" db:"updated_at"`
}

// Address represents a physical address
type Address struct {
	Line1      string  `json:"line1"`
	Line2      *string `json:"line2,omitempty"`
	City       string  `json:"city"`
	State      string  `json:"state"`
	PostalCode string  `json:"postal_code"`
	Country    string  `json:"country"`
}

// Department represents a company department
type Department struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	CorporateAccountID uuid.UUID `json:"corporate_account_id" db:"corporate_account_id"`
	Name              string     `json:"name" db:"name"`
	Code              *string    `json:"code,omitempty" db:"code"`
	ManagerID         *uuid.UUID `json:"manager_id,omitempty" db:"manager_id"`
	BudgetMonthly     *float64   `json:"budget_monthly,omitempty" db:"budget_monthly"`
	BudgetUsed        float64    `json:"budget_used" db:"budget_used"`
	IsActive          bool       `json:"is_active" db:"is_active"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// CorporateEmployee represents an employee in a corporate account
type CorporateEmployee struct {
	ID                uuid.UUID    `json:"id" db:"id"`
	CorporateAccountID uuid.UUID   `json:"corporate_account_id" db:"corporate_account_id"`
	UserID            uuid.UUID    `json:"user_id" db:"user_id"` // Links to rider
	DepartmentID      *uuid.UUID   `json:"department_id,omitempty" db:"department_id"`
	Role              EmployeeRole `json:"role" db:"role"`

	// Employee Details
	EmployeeID        *string      `json:"employee_id,omitempty" db:"employee_id"` // Company's internal ID
	Email             string       `json:"email" db:"email"`
	FirstName         string       `json:"first_name" db:"first_name"`
	LastName          string       `json:"last_name" db:"last_name"`
	JobTitle          *string      `json:"job_title,omitempty" db:"job_title"`

	// Limits
	MonthlyLimit      *float64     `json:"monthly_limit,omitempty" db:"monthly_limit"`
	PerRideLimit      *float64     `json:"per_ride_limit,omitempty" db:"per_ride_limit"`
	MonthlyUsed       float64      `json:"monthly_used" db:"monthly_used"`

	// Settings
	RequireApproval   bool         `json:"require_approval" db:"require_approval"`
	DefaultCostCenter *string      `json:"default_cost_center,omitempty" db:"default_cost_center"`

	IsActive          bool         `json:"is_active" db:"is_active"`
	InvitedAt         *time.Time   `json:"invited_at,omitempty" db:"invited_at"`
	JoinedAt          *time.Time   `json:"joined_at,omitempty" db:"joined_at"`
	CreatedAt         time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at" db:"updated_at"`
}

// RidePolicy represents rules for corporate rides
type RidePolicy struct {
	ID                uuid.UUID   `json:"id" db:"id"`
	CorporateAccountID uuid.UUID  `json:"corporate_account_id" db:"corporate_account_id"`
	DepartmentID      *uuid.UUID  `json:"department_id,omitempty" db:"department_id"` // nil = applies to all
	Name              string      `json:"name" db:"name"`
	Description       *string     `json:"description,omitempty" db:"description"`
	PolicyType        PolicyType  `json:"policy_type" db:"policy_type"`
	Rules             PolicyRules `json:"rules"`
	Priority          int         `json:"priority" db:"priority"` // Higher = more important
	IsActive          bool        `json:"is_active" db:"is_active"`
	CreatedAt         time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at" db:"updated_at"`
}

// PolicyRules contains the specific rules for a policy
type PolicyRules struct {
	// Time restrictions
	AllowedDays       []string    `json:"allowed_days,omitempty"`       // ["monday", "tuesday", ...]
	AllowedStartTime  *string     `json:"allowed_start_time,omitempty"` // "06:00"
	AllowedEndTime    *string     `json:"allowed_end_time,omitempty"`   // "22:00"

	// Location restrictions
	AllowedPickupZones  []string  `json:"allowed_pickup_zones,omitempty"`  // H3 cells or named zones
	AllowedDropoffZones []string  `json:"allowed_dropoff_zones,omitempty"`
	BlockedLocations    []string  `json:"blocked_locations,omitempty"`

	// Amount limits
	MaxAmountPerRide    *float64  `json:"max_amount_per_ride,omitempty"`
	MaxAmountPerDay     *float64  `json:"max_amount_per_day,omitempty"`
	MaxAmountPerWeek    *float64  `json:"max_amount_per_week,omitempty"`
	MaxAmountPerMonth   *float64  `json:"max_amount_per_month,omitempty"`

	// Ride type restrictions
	AllowedRideTypes    []string  `json:"allowed_ride_types,omitempty"` // ["economy", "premium"]
	BlockedRideTypes    []string  `json:"blocked_ride_types,omitempty"`

	// Approval thresholds
	ApprovalThreshold   *float64  `json:"approval_threshold,omitempty"` // Require approval above this amount
}

// CorporateRide represents a ride taken under a corporate account
type CorporateRide struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	RideID            uuid.UUID  `json:"ride_id" db:"ride_id"` // Links to rides table
	CorporateAccountID uuid.UUID `json:"corporate_account_id" db:"corporate_account_id"`
	EmployeeID        uuid.UUID  `json:"employee_id" db:"employee_id"`
	DepartmentID      *uuid.UUID `json:"department_id,omitempty" db:"department_id"`

	// Expense tracking
	CostCenter        *string    `json:"cost_center,omitempty" db:"cost_center"`
	ProjectCode       *string    `json:"project_code,omitempty" db:"project_code"`
	Purpose           *string    `json:"purpose,omitempty" db:"purpose"`
	Notes             *string    `json:"notes,omitempty" db:"notes"`

	// Pricing
	OriginalFare      float64    `json:"original_fare" db:"original_fare"`
	DiscountAmount    float64    `json:"discount_amount" db:"discount_amount"`
	FinalFare         float64    `json:"final_fare" db:"final_fare"`

	// Approval
	RequiresApproval  bool       `json:"requires_approval" db:"requires_approval"`
	ApprovalStatus    *string    `json:"approval_status,omitempty" db:"approval_status"` // pending, approved, rejected
	ApprovedBy        *uuid.UUID `json:"approved_by,omitempty" db:"approved_by"`
	ApprovedAt        *time.Time `json:"approved_at,omitempty" db:"approved_at"`

	// Billing
	InvoiceID         *uuid.UUID `json:"invoice_id,omitempty" db:"invoice_id"`
	BilledAt          *time.Time `json:"billed_at,omitempty" db:"billed_at"`

	// Export tracking
	ExportedToExpense bool       `json:"exported_to_expense" db:"exported_to_expense"`
	ExportedAt        *time.Time `json:"exported_at,omitempty" db:"exported_at"`

	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// CorporateInvoice represents a billing invoice
type CorporateInvoice struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	CorporateAccountID uuid.UUID `json:"corporate_account_id" db:"corporate_account_id"`
	InvoiceNumber     string     `json:"invoice_number" db:"invoice_number"`

	// Period
	PeriodStart       time.Time  `json:"period_start" db:"period_start"`
	PeriodEnd         time.Time  `json:"period_end" db:"period_end"`

	// Amounts
	Subtotal          float64    `json:"subtotal" db:"subtotal"`
	DiscountTotal     float64    `json:"discount_total" db:"discount_total"`
	TaxAmount         float64    `json:"tax_amount" db:"tax_amount"`
	TotalAmount       float64    `json:"total_amount" db:"total_amount"`

	// Status
	Status            string     `json:"status" db:"status"` // draft, sent, paid, overdue, cancelled
	DueDate           time.Time  `json:"due_date" db:"due_date"`
	PaidAt            *time.Time `json:"paid_at,omitempty" db:"paid_at"`
	PaidAmount        float64    `json:"paid_amount" db:"paid_amount"`

	// Metadata
	RideCount         int        `json:"ride_count" db:"ride_count"`
	PDFUrl            *string    `json:"pdf_url,omitempty" db:"pdf_url"`

	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// CostCenter represents a cost center for expense tracking
type CostCenter struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	CorporateAccountID uuid.UUID `json:"corporate_account_id" db:"corporate_account_id"`
	Code              string     `json:"code" db:"code"`
	Name              string     `json:"name" db:"name"`
	Description       *string    `json:"description,omitempty" db:"description"`
	IsActive          bool       `json:"is_active" db:"is_active"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// CreateAccountRequest represents a request to create a corporate account
type CreateAccountRequest struct {
	Name           string       `json:"name" binding:"required"`
	LegalName      string       `json:"legal_name" binding:"required"`
	TaxID          *string      `json:"tax_id,omitempty"`
	PrimaryEmail   string       `json:"primary_email" binding:"required,email"`
	PrimaryPhone   *string      `json:"primary_phone,omitempty"`
	BillingEmail   string       `json:"billing_email" binding:"required,email"`
	Address        *Address     `json:"address,omitempty"`
	BillingCycle   BillingCycle `json:"billing_cycle" binding:"required"`
	PaymentTermDays int         `json:"payment_term_days"`
	Industry       *string      `json:"industry,omitempty"`
	CompanySize    *string      `json:"company_size,omitempty"`
}

// InviteEmployeeRequest represents a request to invite an employee
type InviteEmployeeRequest struct {
	Email          string       `json:"email" binding:"required,email"`
	FirstName      string       `json:"first_name" binding:"required"`
	LastName       string       `json:"last_name" binding:"required"`
	EmployeeID     *string      `json:"employee_id,omitempty"`
	DepartmentID   *uuid.UUID   `json:"department_id,omitempty"`
	Role           EmployeeRole `json:"role"`
	JobTitle       *string      `json:"job_title,omitempty"`
	MonthlyLimit   *float64     `json:"monthly_limit,omitempty"`
	PerRideLimit   *float64     `json:"per_ride_limit,omitempty"`
}

// BookCorporateRideRequest represents a request to book a corporate ride
type BookCorporateRideRequest struct {
	PickupLocation  Location   `json:"pickup_location" binding:"required"`
	DropoffLocation Location   `json:"dropoff_location" binding:"required"`
	RideType        string     `json:"ride_type" binding:"required"`
	CostCenter      *string    `json:"cost_center,omitempty"`
	ProjectCode     *string    `json:"project_code,omitempty"`
	Purpose         *string    `json:"purpose,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	ScheduledTime   *time.Time `json:"scheduled_time,omitempty"`
}

// Location represents a geographic point
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
}

// AccountDashboardResponse represents the corporate dashboard data
type AccountDashboardResponse struct {
	Account          *CorporateAccount      `json:"account"`
	CurrentPeriod    *PeriodStats           `json:"current_period"`
	EmployeeCount    int                    `json:"employee_count"`
	ActiveEmployees  int                    `json:"active_employees"`
	DepartmentCount  int                    `json:"department_count"`
	RecentRides      []CorporateRideDetail  `json:"recent_rides"`
	TopSpenders      []EmployeeSpending     `json:"top_spenders"`
	DepartmentUsage  []DepartmentUsage      `json:"department_usage"`
	PendingApprovals int                    `json:"pending_approvals"`
}

// PeriodStats represents statistics for a billing period
type PeriodStats struct {
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	TotalRides     int       `json:"total_rides"`
	TotalSpent     float64   `json:"total_spent"`
	TotalSaved     float64   `json:"total_saved"` // From corporate discount
	AvgRideCost    float64   `json:"avg_ride_cost"`
	BudgetUsed     float64   `json:"budget_used_percent"`
}

// CorporateRideDetail represents detailed ride information
type CorporateRideDetail struct {
	CorporateRide
	EmployeeName   string     `json:"employee_name"`
	DepartmentName *string    `json:"department_name,omitempty"`
	PickupAddress  string     `json:"pickup_address"`
	DropoffAddress string     `json:"dropoff_address"`
	RideDate       time.Time  `json:"ride_date"`
	RideStatus     string     `json:"ride_status"`
}

// EmployeeSpending represents spending by employee
type EmployeeSpending struct {
	EmployeeID   uuid.UUID `json:"employee_id"`
	EmployeeName string    `json:"employee_name"`
	RideCount    int       `json:"ride_count"`
	TotalSpent   float64   `json:"total_spent"`
	AvgPerRide   float64   `json:"avg_per_ride"`
}

// DepartmentUsage represents usage by department
type DepartmentUsage struct {
	DepartmentID   uuid.UUID `json:"department_id"`
	DepartmentName string    `json:"department_name"`
	RideCount      int       `json:"ride_count"`
	TotalSpent     float64   `json:"total_spent"`
	BudgetUsed     float64   `json:"budget_used_percent"`
}

// PolicyCheckResult represents the result of checking a ride against policies
type PolicyCheckResult struct {
	Allowed       bool              `json:"allowed"`
	Violations    []PolicyViolation `json:"violations,omitempty"`
	RequiresApproval bool           `json:"requires_approval"`
	ApprovalReason   *string        `json:"approval_reason,omitempty"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	PolicyID    uuid.UUID `json:"policy_id"`
	PolicyName  string    `json:"policy_name"`
	Reason      string    `json:"reason"`
	Severity    string    `json:"severity"` // warning, block
}

// ExpenseExportRequest represents a request to export rides to expense system
type ExpenseExportRequest struct {
	StartDate   time.Time   `json:"start_date" binding:"required"`
	EndDate     time.Time   `json:"end_date" binding:"required"`
	Format      string      `json:"format"` // csv, json, concur, sap
	EmployeeIDs []uuid.UUID `json:"employee_ids,omitempty"` // Filter by employees
}

// ExpenseExportResponse represents the export result
type ExpenseExportResponse struct {
	ExportID     uuid.UUID `json:"export_id"`
	RideCount    int       `json:"ride_count"`
	TotalAmount  float64   `json:"total_amount"`
	DownloadURL  string    `json:"download_url"`
	ExpiresAt    time.Time `json:"expires_at"`
}
