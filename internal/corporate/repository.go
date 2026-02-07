package corporate

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// Repository handles corporate account database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new corporate repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// CORPORATE ACCOUNT OPERATIONS
// ========================================

// CreateAccount creates a new corporate account
func (r *Repository) CreateAccount(ctx context.Context, account *CorporateAccount) error {
	addressJSON, err := json.Marshal(account.Address)
	if err != nil {
		logger.Get().Warn("Failed to marshal corporate account address",
			zap.String("account_id", account.ID.String()),
			zap.Error(err),
		)
		addressJSON = []byte("{}")
	}

	query := `
		INSERT INTO corporate_accounts (
			id, name, legal_name, tax_id, status,
			primary_email, primary_phone, billing_email, address,
			billing_cycle, payment_term_days, credit_limit, current_balance,
			discount_percent, custom_rates,
			require_approval, require_cost_center, require_project_code, allow_personal_rides,
			expense_system_id, sso_enabled, sso_provider,
			logo_url, industry, company_size,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)
	`

	_, execErr := r.db.Exec(ctx, query,
		account.ID, account.Name, account.LegalName, account.TaxID, account.Status,
		account.PrimaryEmail, account.PrimaryPhone, account.BillingEmail, addressJSON,
		account.BillingCycle, account.PaymentTermDays, account.CreditLimit, account.CurrentBalance,
		account.DiscountPercent, account.CustomRates,
		account.RequireApproval, account.RequireCostCenter, account.RequireProjectCode, account.AllowPersonalRides,
		account.ExpenseSystemID, account.SSOEnabled, account.SSOProvider,
		account.LogoURL, account.Industry, account.CompanySize,
		account.CreatedAt, account.UpdatedAt,
	)
	return execErr
}

// GetAccount gets a corporate account by ID
func (r *Repository) GetAccount(ctx context.Context, accountID uuid.UUID) (*CorporateAccount, error) {
	query := `
		SELECT id, name, legal_name, tax_id, status,
			primary_email, primary_phone, billing_email, address,
			billing_cycle, payment_term_days, credit_limit, current_balance,
			discount_percent, custom_rates,
			require_approval, require_cost_center, require_project_code, allow_personal_rides,
			expense_system_id, sso_enabled, sso_provider,
			logo_url, industry, company_size,
			created_at, updated_at
		FROM corporate_accounts
		WHERE id = $1
	`

	var account CorporateAccount
	var addressJSON []byte
	err := r.db.QueryRow(ctx, query, accountID).Scan(
		&account.ID, &account.Name, &account.LegalName, &account.TaxID, &account.Status,
		&account.PrimaryEmail, &account.PrimaryPhone, &account.BillingEmail, &addressJSON,
		&account.BillingCycle, &account.PaymentTermDays, &account.CreditLimit, &account.CurrentBalance,
		&account.DiscountPercent, &account.CustomRates,
		&account.RequireApproval, &account.RequireCostCenter, &account.RequireProjectCode, &account.AllowPersonalRides,
		&account.ExpenseSystemID, &account.SSOEnabled, &account.SSOProvider,
		&account.LogoURL, &account.Industry, &account.CompanySize,
		&account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if addressJSON != nil {
		if err := json.Unmarshal(addressJSON, &account.Address); err != nil {
			logger.Get().Warn("Failed to unmarshal corporate account address",
				zap.String("account_id", account.ID.String()),
				zap.Error(err),
			)
		}
	}

	return &account, nil
}

// UpdateAccount updates a corporate account
func (r *Repository) UpdateAccount(ctx context.Context, account *CorporateAccount) error {
	addressJSON, err := json.Marshal(account.Address)
	if err != nil {
		logger.Get().Warn("Failed to marshal corporate account address for update",
			zap.String("account_id", account.ID.String()),
			zap.Error(err),
		)
		addressJSON = []byte("{}")
	}

	query := `
		UPDATE corporate_accounts SET
			name = $1, legal_name = $2, tax_id = $3, status = $4,
			primary_email = $5, primary_phone = $6, billing_email = $7, address = $8,
			billing_cycle = $9, payment_term_days = $10, credit_limit = $11,
			discount_percent = $12, custom_rates = $13,
			require_approval = $14, require_cost_center = $15, require_project_code = $16, allow_personal_rides = $17,
			expense_system_id = $18, sso_enabled = $19, sso_provider = $20,
			logo_url = $21, industry = $22, company_size = $23,
			updated_at = NOW()
		WHERE id = $24
	`

	_, execErr := r.db.Exec(ctx, query,
		account.Name, account.LegalName, account.TaxID, account.Status,
		account.PrimaryEmail, account.PrimaryPhone, account.BillingEmail, addressJSON,
		account.BillingCycle, account.PaymentTermDays, account.CreditLimit,
		account.DiscountPercent, account.CustomRates,
		account.RequireApproval, account.RequireCostCenter, account.RequireProjectCode, account.AllowPersonalRides,
		account.ExpenseSystemID, account.SSOEnabled, account.SSOProvider,
		account.LogoURL, account.Industry, account.CompanySize,
		account.ID,
	)
	return execErr
}

// UpdateAccountStatus updates account status
func (r *Repository) UpdateAccountStatus(ctx context.Context, accountID uuid.UUID, status AccountStatus) error {
	query := `UPDATE corporate_accounts SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, accountID)
	return err
}

// UpdateAccountBalance updates account balance
func (r *Repository) UpdateAccountBalance(ctx context.Context, accountID uuid.UUID, amount float64) error {
	query := `UPDATE corporate_accounts SET current_balance = current_balance + $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, amount, accountID)
	return err
}

// ListAccounts lists corporate accounts with filters
func (r *Repository) ListAccounts(ctx context.Context, status *AccountStatus, limit, offset int) ([]*CorporateAccount, error) {
	query := `
		SELECT id, name, legal_name, tax_id, status,
			primary_email, primary_phone, billing_email, address,
			billing_cycle, payment_term_days, credit_limit, current_balance,
			discount_percent, custom_rates,
			require_approval, require_cost_center, require_project_code, allow_personal_rides,
			expense_system_id, sso_enabled, sso_provider,
			logo_url, industry, company_size,
			created_at, updated_at
		FROM corporate_accounts
		WHERE ($1::text IS NULL OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var statusStr *string
	if status != nil {
		s := string(*status)
		statusStr = &s
	}

	rows, err := r.db.Query(ctx, query, statusStr, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*CorporateAccount
	for rows.Next() {
		var account CorporateAccount
		var addressJSON []byte
		err := rows.Scan(
			&account.ID, &account.Name, &account.LegalName, &account.TaxID, &account.Status,
			&account.PrimaryEmail, &account.PrimaryPhone, &account.BillingEmail, &addressJSON,
			&account.BillingCycle, &account.PaymentTermDays, &account.CreditLimit, &account.CurrentBalance,
			&account.DiscountPercent, &account.CustomRates,
			&account.RequireApproval, &account.RequireCostCenter, &account.RequireProjectCode, &account.AllowPersonalRides,
			&account.ExpenseSystemID, &account.SSOEnabled, &account.SSOProvider,
			&account.LogoURL, &account.Industry, &account.CompanySize,
			&account.CreatedAt, &account.UpdatedAt,
		)
		if err != nil {
			continue
		}
		if addressJSON != nil {
			if unmarshalErr := json.Unmarshal(addressJSON, &account.Address); unmarshalErr != nil {
				logger.Get().Warn("Failed to unmarshal corporate account address in list",
					zap.String("account_id", account.ID.String()),
					zap.Error(unmarshalErr),
				)
			}
		}
		accounts = append(accounts, &account)
	}

	return accounts, nil
}

// ========================================
// DEPARTMENT OPERATIONS
// ========================================

// CreateDepartment creates a new department
func (r *Repository) CreateDepartment(ctx context.Context, dept *Department) error {
	query := `
		INSERT INTO corporate_departments (
			id, corporate_account_id, name, code, manager_id,
			budget_monthly, budget_used, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.Exec(ctx, query,
		dept.ID, dept.CorporateAccountID, dept.Name, dept.Code, dept.ManagerID,
		dept.BudgetMonthly, dept.BudgetUsed, dept.IsActive, dept.CreatedAt, dept.UpdatedAt,
	)
	return err
}

// GetDepartment gets a department by ID
func (r *Repository) GetDepartment(ctx context.Context, deptID uuid.UUID) (*Department, error) {
	query := `
		SELECT id, corporate_account_id, name, code, manager_id,
			budget_monthly, budget_used, is_active, created_at, updated_at
		FROM corporate_departments
		WHERE id = $1
	`

	var dept Department
	err := r.db.QueryRow(ctx, query, deptID).Scan(
		&dept.ID, &dept.CorporateAccountID, &dept.Name, &dept.Code, &dept.ManagerID,
		&dept.BudgetMonthly, &dept.BudgetUsed, &dept.IsActive, &dept.CreatedAt, &dept.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &dept, nil
}

// ListDepartments lists departments for an account
func (r *Repository) ListDepartments(ctx context.Context, accountID uuid.UUID) ([]*Department, error) {
	query := `
		SELECT id, corporate_account_id, name, code, manager_id,
			budget_monthly, budget_used, is_active, created_at, updated_at
		FROM corporate_departments
		WHERE corporate_account_id = $1 AND is_active = true
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var depts []*Department
	for rows.Next() {
		var dept Department
		err := rows.Scan(
			&dept.ID, &dept.CorporateAccountID, &dept.Name, &dept.Code, &dept.ManagerID,
			&dept.BudgetMonthly, &dept.BudgetUsed, &dept.IsActive, &dept.CreatedAt, &dept.UpdatedAt,
		)
		if err != nil {
			continue
		}
		depts = append(depts, &dept)
	}
	return depts, nil
}

// UpdateDepartmentBudget updates department budget usage
func (r *Repository) UpdateDepartmentBudget(ctx context.Context, deptID uuid.UUID, amount float64) error {
	query := `UPDATE corporate_departments SET budget_used = budget_used + $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, amount, deptID)
	return err
}

// ResetDepartmentBudgets resets all department budgets (monthly reset)
func (r *Repository) ResetDepartmentBudgets(ctx context.Context, accountID uuid.UUID) error {
	query := `UPDATE corporate_departments SET budget_used = 0, updated_at = NOW() WHERE corporate_account_id = $1`
	_, err := r.db.Exec(ctx, query, accountID)
	return err
}

// ========================================
// EMPLOYEE OPERATIONS
// ========================================

// CreateEmployee creates a new employee
func (r *Repository) CreateEmployee(ctx context.Context, emp *CorporateEmployee) error {
	query := `
		INSERT INTO corporate_employees (
			id, corporate_account_id, user_id, department_id, role,
			employee_id, email, first_name, last_name, job_title,
			monthly_limit, per_ride_limit, monthly_used,
			require_approval, default_cost_center,
			is_active, invited_at, joined_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`
	_, err := r.db.Exec(ctx, query,
		emp.ID, emp.CorporateAccountID, emp.UserID, emp.DepartmentID, emp.Role,
		emp.EmployeeID, emp.Email, emp.FirstName, emp.LastName, emp.JobTitle,
		emp.MonthlyLimit, emp.PerRideLimit, emp.MonthlyUsed,
		emp.RequireApproval, emp.DefaultCostCenter,
		emp.IsActive, emp.InvitedAt, emp.JoinedAt, emp.CreatedAt, emp.UpdatedAt,
	)
	return err
}

// GetEmployee gets an employee by ID
func (r *Repository) GetEmployee(ctx context.Context, empID uuid.UUID) (*CorporateEmployee, error) {
	query := `
		SELECT id, corporate_account_id, user_id, department_id, role,
			employee_id, email, first_name, last_name, job_title,
			monthly_limit, per_ride_limit, monthly_used,
			require_approval, default_cost_center,
			is_active, invited_at, joined_at, created_at, updated_at
		FROM corporate_employees
		WHERE id = $1
	`

	var emp CorporateEmployee
	err := r.db.QueryRow(ctx, query, empID).Scan(
		&emp.ID, &emp.CorporateAccountID, &emp.UserID, &emp.DepartmentID, &emp.Role,
		&emp.EmployeeID, &emp.Email, &emp.FirstName, &emp.LastName, &emp.JobTitle,
		&emp.MonthlyLimit, &emp.PerRideLimit, &emp.MonthlyUsed,
		&emp.RequireApproval, &emp.DefaultCostCenter,
		&emp.IsActive, &emp.InvitedAt, &emp.JoinedAt, &emp.CreatedAt, &emp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &emp, nil
}

// GetEmployeeByUserID gets an employee by user ID
func (r *Repository) GetEmployeeByUserID(ctx context.Context, userID uuid.UUID) (*CorporateEmployee, error) {
	query := `
		SELECT id, corporate_account_id, user_id, department_id, role,
			employee_id, email, first_name, last_name, job_title,
			monthly_limit, per_ride_limit, monthly_used,
			require_approval, default_cost_center,
			is_active, invited_at, joined_at, created_at, updated_at
		FROM corporate_employees
		WHERE user_id = $1 AND is_active = true
	`

	var emp CorporateEmployee
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&emp.ID, &emp.CorporateAccountID, &emp.UserID, &emp.DepartmentID, &emp.Role,
		&emp.EmployeeID, &emp.Email, &emp.FirstName, &emp.LastName, &emp.JobTitle,
		&emp.MonthlyLimit, &emp.PerRideLimit, &emp.MonthlyUsed,
		&emp.RequireApproval, &emp.DefaultCostCenter,
		&emp.IsActive, &emp.InvitedAt, &emp.JoinedAt, &emp.CreatedAt, &emp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &emp, nil
}

// GetEmployeeByEmail gets an employee by email
func (r *Repository) GetEmployeeByEmail(ctx context.Context, accountID uuid.UUID, email string) (*CorporateEmployee, error) {
	query := `
		SELECT id, corporate_account_id, user_id, department_id, role,
			employee_id, email, first_name, last_name, job_title,
			monthly_limit, per_ride_limit, monthly_used,
			require_approval, default_cost_center,
			is_active, invited_at, joined_at, created_at, updated_at
		FROM corporate_employees
		WHERE corporate_account_id = $1 AND email = $2
	`

	var emp CorporateEmployee
	err := r.db.QueryRow(ctx, query, accountID, email).Scan(
		&emp.ID, &emp.CorporateAccountID, &emp.UserID, &emp.DepartmentID, &emp.Role,
		&emp.EmployeeID, &emp.Email, &emp.FirstName, &emp.LastName, &emp.JobTitle,
		&emp.MonthlyLimit, &emp.PerRideLimit, &emp.MonthlyUsed,
		&emp.RequireApproval, &emp.DefaultCostCenter,
		&emp.IsActive, &emp.InvitedAt, &emp.JoinedAt, &emp.CreatedAt, &emp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &emp, nil
}

// ListEmployees lists employees for an account
func (r *Repository) ListEmployees(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*CorporateEmployee, error) {
	query := `
		SELECT id, corporate_account_id, user_id, department_id, role,
			employee_id, email, first_name, last_name, job_title,
			monthly_limit, per_ride_limit, monthly_used,
			require_approval, default_cost_center,
			is_active, invited_at, joined_at, created_at, updated_at
		FROM corporate_employees
		WHERE corporate_account_id = $1
		ORDER BY last_name, first_name ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []*CorporateEmployee
	for rows.Next() {
		var emp CorporateEmployee
		err := rows.Scan(
			&emp.ID, &emp.CorporateAccountID, &emp.UserID, &emp.DepartmentID, &emp.Role,
			&emp.EmployeeID, &emp.Email, &emp.FirstName, &emp.LastName, &emp.JobTitle,
			&emp.MonthlyLimit, &emp.PerRideLimit, &emp.MonthlyUsed,
			&emp.RequireApproval, &emp.DefaultCostCenter,
			&emp.IsActive, &emp.InvitedAt, &emp.JoinedAt, &emp.CreatedAt, &emp.UpdatedAt,
		)
		if err != nil {
			continue
		}
		employees = append(employees, &emp)
	}
	return employees, nil
}

// UpdateEmployeeUsage updates employee's monthly usage
func (r *Repository) UpdateEmployeeUsage(ctx context.Context, empID uuid.UUID, amount float64) error {
	query := `UPDATE corporate_employees SET monthly_used = monthly_used + $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, amount, empID)
	return err
}

// ResetEmployeeUsage resets all employee monthly usage (monthly reset)
func (r *Repository) ResetEmployeeUsage(ctx context.Context, accountID uuid.UUID) error {
	query := `UPDATE corporate_employees SET monthly_used = 0, updated_at = NOW() WHERE corporate_account_id = $1`
	_, err := r.db.Exec(ctx, query, accountID)
	return err
}

// GetEmployeeCount gets the count of employees for an account
func (r *Repository) GetEmployeeCount(ctx context.Context, accountID uuid.UUID, activeOnly bool) (int, error) {
	query := `SELECT COUNT(*) FROM corporate_employees WHERE corporate_account_id = $1`
	if activeOnly {
		query += ` AND is_active = true`
	}

	var count int
	err := r.db.QueryRow(ctx, query, accountID).Scan(&count)
	return count, err
}

// ========================================
// POLICY OPERATIONS
// ========================================

// CreatePolicy creates a new policy
func (r *Repository) CreatePolicy(ctx context.Context, policy *RidePolicy) error {
	rulesJSON, err := json.Marshal(policy.Rules)
	if err != nil {
		logger.Get().Warn("Failed to marshal policy rules",
			zap.String("policy_id", policy.ID.String()),
			zap.Error(err),
		)
		rulesJSON = []byte("{}")
	}

	query := `
		INSERT INTO corporate_policies (
			id, corporate_account_id, department_id, name, description,
			policy_type, rules, priority, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, execErr := r.db.Exec(ctx, query,
		policy.ID, policy.CorporateAccountID, policy.DepartmentID, policy.Name, policy.Description,
		policy.PolicyType, rulesJSON, policy.Priority, policy.IsActive, policy.CreatedAt, policy.UpdatedAt,
	)
	return execErr
}

// GetPolicies gets policies for an account/department
func (r *Repository) GetPolicies(ctx context.Context, accountID uuid.UUID, departmentID *uuid.UUID) ([]*RidePolicy, error) {
	query := `
		SELECT id, corporate_account_id, department_id, name, description,
			policy_type, rules, priority, is_active, created_at, updated_at
		FROM corporate_policies
		WHERE corporate_account_id = $1
			AND (department_id IS NULL OR department_id = $2)
			AND is_active = true
		ORDER BY priority DESC
	`

	rows, err := r.db.Query(ctx, query, accountID, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*RidePolicy
	for rows.Next() {
		var policy RidePolicy
		var rulesJSON []byte
		err := rows.Scan(
			&policy.ID, &policy.CorporateAccountID, &policy.DepartmentID, &policy.Name, &policy.Description,
			&policy.PolicyType, &rulesJSON, &policy.Priority, &policy.IsActive, &policy.CreatedAt, &policy.UpdatedAt,
		)
		if err != nil {
			continue
		}
		if rulesJSON != nil {
			if unmarshalErr := json.Unmarshal(rulesJSON, &policy.Rules); unmarshalErr != nil {
				logger.Get().Warn("Failed to unmarshal policy rules",
					zap.String("policy_id", policy.ID.String()),
					zap.Error(unmarshalErr),
				)
			}
		}
		policies = append(policies, &policy)
	}
	return policies, nil
}

// ========================================
// CORPORATE RIDE OPERATIONS
// ========================================

// CreateCorporateRide creates a corporate ride record
func (r *Repository) CreateCorporateRide(ctx context.Context, ride *CorporateRide) error {
	query := `
		INSERT INTO corporate_rides (
			id, ride_id, corporate_account_id, employee_id, department_id,
			cost_center, project_code, purpose, notes,
			original_fare, discount_amount, final_fare,
			requires_approval, approval_status, approved_by, approved_at,
			invoice_id, billed_at, exported_to_expense, exported_at,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`
	_, err := r.db.Exec(ctx, query,
		ride.ID, ride.RideID, ride.CorporateAccountID, ride.EmployeeID, ride.DepartmentID,
		ride.CostCenter, ride.ProjectCode, ride.Purpose, ride.Notes,
		ride.OriginalFare, ride.DiscountAmount, ride.FinalFare,
		ride.RequiresApproval, ride.ApprovalStatus, ride.ApprovedBy, ride.ApprovedAt,
		ride.InvoiceID, ride.BilledAt, ride.ExportedToExpense, ride.ExportedAt,
		ride.CreatedAt,
	)
	return err
}

// GetCorporateRide gets a corporate ride by ID
func (r *Repository) GetCorporateRide(ctx context.Context, rideID uuid.UUID) (*CorporateRide, error) {
	query := `
		SELECT id, ride_id, corporate_account_id, employee_id, department_id,
			cost_center, project_code, purpose, notes,
			original_fare, discount_amount, final_fare,
			requires_approval, approval_status, approved_by, approved_at,
			invoice_id, billed_at, exported_to_expense, exported_at,
			created_at
		FROM corporate_rides
		WHERE id = $1
	`

	var ride CorporateRide
	err := r.db.QueryRow(ctx, query, rideID).Scan(
		&ride.ID, &ride.RideID, &ride.CorporateAccountID, &ride.EmployeeID, &ride.DepartmentID,
		&ride.CostCenter, &ride.ProjectCode, &ride.Purpose, &ride.Notes,
		&ride.OriginalFare, &ride.DiscountAmount, &ride.FinalFare,
		&ride.RequiresApproval, &ride.ApprovalStatus, &ride.ApprovedBy, &ride.ApprovedAt,
		&ride.InvoiceID, &ride.BilledAt, &ride.ExportedToExpense, &ride.ExportedAt,
		&ride.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ride, nil
}

// ListCorporateRides lists corporate rides with filters
func (r *Repository) ListCorporateRides(ctx context.Context, accountID uuid.UUID, employeeID *uuid.UUID, startDate, endDate time.Time, limit, offset int) ([]*CorporateRide, error) {
	query := `
		SELECT id, ride_id, corporate_account_id, employee_id, department_id,
			cost_center, project_code, purpose, notes,
			original_fare, discount_amount, final_fare,
			requires_approval, approval_status, approved_by, approved_at,
			invoice_id, billed_at, exported_to_expense, exported_at,
			created_at
		FROM corporate_rides
		WHERE corporate_account_id = $1
			AND ($2::uuid IS NULL OR employee_id = $2)
			AND created_at >= $3 AND created_at <= $4
		ORDER BY created_at DESC
		LIMIT $5 OFFSET $6
	`

	rows, err := r.db.Query(ctx, query, accountID, employeeID, startDate, endDate, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rides []*CorporateRide
	for rows.Next() {
		var ride CorporateRide
		err := rows.Scan(
			&ride.ID, &ride.RideID, &ride.CorporateAccountID, &ride.EmployeeID, &ride.DepartmentID,
			&ride.CostCenter, &ride.ProjectCode, &ride.Purpose, &ride.Notes,
			&ride.OriginalFare, &ride.DiscountAmount, &ride.FinalFare,
			&ride.RequiresApproval, &ride.ApprovalStatus, &ride.ApprovedBy, &ride.ApprovedAt,
			&ride.InvoiceID, &ride.BilledAt, &ride.ExportedToExpense, &ride.ExportedAt,
			&ride.CreatedAt,
		)
		if err != nil {
			continue
		}
		rides = append(rides, &ride)
	}
	return rides, nil
}

// GetPendingApprovals gets rides pending approval
func (r *Repository) GetPendingApprovals(ctx context.Context, accountID uuid.UUID, approverID *uuid.UUID) ([]*CorporateRide, error) {
	query := `
		SELECT cr.id, cr.ride_id, cr.corporate_account_id, cr.employee_id, cr.department_id,
			cr.cost_center, cr.project_code, cr.purpose, cr.notes,
			cr.original_fare, cr.discount_amount, cr.final_fare,
			cr.requires_approval, cr.approval_status, cr.approved_by, cr.approved_at,
			cr.invoice_id, cr.billed_at, cr.exported_to_expense, cr.exported_at,
			cr.created_at
		FROM corporate_rides cr
		WHERE cr.corporate_account_id = $1
			AND cr.requires_approval = true
			AND cr.approval_status = 'pending'
		ORDER BY cr.created_at ASC
	`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rides []*CorporateRide
	for rows.Next() {
		var ride CorporateRide
		err := rows.Scan(
			&ride.ID, &ride.RideID, &ride.CorporateAccountID, &ride.EmployeeID, &ride.DepartmentID,
			&ride.CostCenter, &ride.ProjectCode, &ride.Purpose, &ride.Notes,
			&ride.OriginalFare, &ride.DiscountAmount, &ride.FinalFare,
			&ride.RequiresApproval, &ride.ApprovalStatus, &ride.ApprovedBy, &ride.ApprovedAt,
			&ride.InvoiceID, &ride.BilledAt, &ride.ExportedToExpense, &ride.ExportedAt,
			&ride.CreatedAt,
		)
		if err != nil {
			continue
		}
		rides = append(rides, &ride)
	}
	return rides, nil
}

// ApproveRide approves or rejects a ride
func (r *Repository) ApproveRide(ctx context.Context, rideID, approverID uuid.UUID, approved bool) error {
	status := "approved"
	if !approved {
		status = "rejected"
	}

	query := `
		UPDATE corporate_rides
		SET approval_status = $1, approved_by = $2, approved_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, status, approverID, rideID)
	return err
}

// ========================================
// INVOICE OPERATIONS
// ========================================

// CreateInvoice creates a new invoice
func (r *Repository) CreateInvoice(ctx context.Context, invoice *CorporateInvoice) error {
	query := `
		INSERT INTO corporate_invoices (
			id, corporate_account_id, invoice_number,
			period_start, period_end,
			subtotal, discount_total, tax_amount, total_amount,
			status, due_date, paid_at, paid_amount,
			ride_count, pdf_url,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	_, err := r.db.Exec(ctx, query,
		invoice.ID, invoice.CorporateAccountID, invoice.InvoiceNumber,
		invoice.PeriodStart, invoice.PeriodEnd,
		invoice.Subtotal, invoice.DiscountTotal, invoice.TaxAmount, invoice.TotalAmount,
		invoice.Status, invoice.DueDate, invoice.PaidAt, invoice.PaidAmount,
		invoice.RideCount, invoice.PDFUrl,
		invoice.CreatedAt, invoice.UpdatedAt,
	)
	return err
}

// GetInvoice gets an invoice by ID
func (r *Repository) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*CorporateInvoice, error) {
	query := `
		SELECT id, corporate_account_id, invoice_number,
			period_start, period_end,
			subtotal, discount_total, tax_amount, total_amount,
			status, due_date, paid_at, paid_amount,
			ride_count, pdf_url,
			created_at, updated_at
		FROM corporate_invoices
		WHERE id = $1
	`

	var invoice CorporateInvoice
	err := r.db.QueryRow(ctx, query, invoiceID).Scan(
		&invoice.ID, &invoice.CorporateAccountID, &invoice.InvoiceNumber,
		&invoice.PeriodStart, &invoice.PeriodEnd,
		&invoice.Subtotal, &invoice.DiscountTotal, &invoice.TaxAmount, &invoice.TotalAmount,
		&invoice.Status, &invoice.DueDate, &invoice.PaidAt, &invoice.PaidAmount,
		&invoice.RideCount, &invoice.PDFUrl,
		&invoice.CreatedAt, &invoice.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

// ListInvoices lists invoices for an account
func (r *Repository) ListInvoices(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*CorporateInvoice, error) {
	query := `
		SELECT id, corporate_account_id, invoice_number,
			period_start, period_end,
			subtotal, discount_total, tax_amount, total_amount,
			status, due_date, paid_at, paid_amount,
			ride_count, pdf_url,
			created_at, updated_at
		FROM corporate_invoices
		WHERE corporate_account_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*CorporateInvoice
	for rows.Next() {
		var invoice CorporateInvoice
		err := rows.Scan(
			&invoice.ID, &invoice.CorporateAccountID, &invoice.InvoiceNumber,
			&invoice.PeriodStart, &invoice.PeriodEnd,
			&invoice.Subtotal, &invoice.DiscountTotal, &invoice.TaxAmount, &invoice.TotalAmount,
			&invoice.Status, &invoice.DueDate, &invoice.PaidAt, &invoice.PaidAmount,
			&invoice.RideCount, &invoice.PDFUrl,
			&invoice.CreatedAt, &invoice.UpdatedAt,
		)
		if err != nil {
			continue
		}
		invoices = append(invoices, &invoice)
	}
	return invoices, nil
}

// ========================================
// STATISTICS
// ========================================

// GetPeriodStats gets statistics for a billing period
func (r *Repository) GetPeriodStats(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time) (*PeriodStats, error) {
	query := `
		SELECT COUNT(*), COALESCE(SUM(final_fare), 0), COALESCE(SUM(discount_amount), 0)
		FROM corporate_rides
		WHERE corporate_account_id = $1
			AND created_at >= $2 AND created_at <= $3
	`

	var rideCount int
	var totalSpent, totalSaved float64
	err := r.db.QueryRow(ctx, query, accountID, startDate, endDate).Scan(&rideCount, &totalSpent, &totalSaved)
	if err != nil {
		return nil, err
	}

	avgCost := 0.0
	if rideCount > 0 {
		avgCost = totalSpent / float64(rideCount)
	}

	return &PeriodStats{
		PeriodStart: startDate,
		PeriodEnd:   endDate,
		TotalRides:  rideCount,
		TotalSpent:  totalSpent,
		TotalSaved:  totalSaved,
		AvgRideCost: avgCost,
	}, nil
}

// GetTopSpenders gets top spending employees
func (r *Repository) GetTopSpenders(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time, limit int) ([]EmployeeSpending, error) {
	query := `
		SELECT ce.id, CONCAT(ce.first_name, ' ', ce.last_name) as name,
			COUNT(cr.id) as ride_count,
			COALESCE(SUM(cr.final_fare), 0) as total_spent
		FROM corporate_employees ce
		LEFT JOIN corporate_rides cr ON ce.id = cr.employee_id
			AND cr.created_at >= $2 AND cr.created_at <= $3
		WHERE ce.corporate_account_id = $1
		GROUP BY ce.id, ce.first_name, ce.last_name
		ORDER BY total_spent DESC
		LIMIT $4
	`

	rows, err := r.db.Query(ctx, query, accountID, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spenders []EmployeeSpending
	for rows.Next() {
		var s EmployeeSpending
		err := rows.Scan(&s.EmployeeID, &s.EmployeeName, &s.RideCount, &s.TotalSpent)
		if err != nil {
			continue
		}
		if s.RideCount > 0 {
			s.AvgPerRide = s.TotalSpent / float64(s.RideCount)
		}
		spenders = append(spenders, s)
	}
	return spenders, nil
}

// GetDepartmentUsage gets usage by department
func (r *Repository) GetDepartmentUsage(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time) ([]DepartmentUsage, error) {
	query := `
		SELECT d.id, d.name,
			COUNT(cr.id) as ride_count,
			COALESCE(SUM(cr.final_fare), 0) as total_spent,
			d.budget_monthly, d.budget_used
		FROM corporate_departments d
		LEFT JOIN corporate_rides cr ON d.id = cr.department_id
			AND cr.created_at >= $2 AND cr.created_at <= $3
		WHERE d.corporate_account_id = $1 AND d.is_active = true
		GROUP BY d.id, d.name, d.budget_monthly, d.budget_used
		ORDER BY total_spent DESC
	`

	rows, err := r.db.Query(ctx, query, accountID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usage []DepartmentUsage
	for rows.Next() {
		var u DepartmentUsage
		var budgetMonthly, budgetUsed *float64
		err := rows.Scan(&u.DepartmentID, &u.DepartmentName, &u.RideCount, &u.TotalSpent, &budgetMonthly, &budgetUsed)
		if err != nil {
			continue
		}
		if budgetMonthly != nil && *budgetMonthly > 0 && budgetUsed != nil {
			u.BudgetUsed = (*budgetUsed / *budgetMonthly) * 100
		}
		usage = append(usage, u)
	}
	return usage, nil
}
