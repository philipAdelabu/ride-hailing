//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/richxcame/ride-hailing/internal/corporate"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

const corporateServiceKey = "corporate"

// CorporateFlowTestSuite tests corporate account flows
type CorporateFlowTestSuite struct {
	suite.Suite
	admin  authSession
	rider  authSession
	driver authSession
}

func TestCorporateFlowSuite(t *testing.T) {
	suite.Run(t, new(CorporateFlowTestSuite))
}

func (s *CorporateFlowTestSuite) SetupSuite() {
	// Ensure all services are started
	if _, ok := services[authServiceKey]; !ok {
		services[authServiceKey] = startAuthService(mustLoadConfig("auth-service"))
	}
	if _, ok := services[ridesServiceKey]; !ok {
		services[ridesServiceKey] = startRidesService(mustLoadConfig("rides-service"))
	}
	if _, ok := services[paymentsServiceKey]; !ok {
		services[paymentsServiceKey] = startPaymentsService(mustLoadConfig("payments-service"))
	}
	if _, ok := services[corporateServiceKey]; !ok {
		services[corporateServiceKey] = startCorporateService()
	}
}

func (s *CorporateFlowTestSuite) SetupTest() {
	truncateCorporateTables(s.T())
	s.admin = registerAndLogin(s.T(), models.RoleAdmin)
	s.rider = registerAndLogin(s.T(), models.RoleRider)
	s.driver = registerAndLogin(s.T(), models.RoleDriver)
}

func startCorporateService() *serviceInstance {
	repo := corporate.NewRepository(dbPool)
	service := corporate.NewService(repo)
	handler := corporate.NewHandler(service)

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.CorrelationID())

	// Public corporate signup
	router.POST("/api/v1/corporate/accounts", handler.CreateAccount)

	// Authenticated corporate routes
	api := router.Group("/api/v1/corporate")
	api.Use(middleware.AuthMiddleware("integration-secret"))
	{
		api.GET("/me", handler.GetMyProfile)
		api.GET("/accounts/:id", handler.GetAccount)
		api.GET("/accounts/:id/dashboard", handler.GetDashboard)
		api.GET("/accounts/:id/departments", handler.ListDepartments)
		api.POST("/accounts/:id/departments", handler.CreateDepartment)
		api.GET("/accounts/:id/employees", handler.ListEmployees)
		api.POST("/accounts/:id/employees", handler.InviteEmployee)
		api.GET("/accounts/:id/rides", handler.ListRides)
		api.GET("/accounts/:id/approvals", handler.GetPendingApprovals)
		api.GET("/accounts/:id/invoices", handler.ListInvoices)
		api.POST("/accounts/:id/invoices", handler.GenerateInvoice)
		api.POST("/accounts/:id/policies", handler.CreatePolicy)
		api.POST("/rides/:id/approve", handler.ApproveRide)
	}

	// Admin routes
	admin := router.Group("/api/v1/admin/corporate")
	admin.Use(middleware.AuthMiddleware("integration-secret"))
	admin.Use(middleware.RequireAdmin())
	{
		admin.GET("/accounts", handler.ListAccounts)
		admin.POST("/accounts/:id/activate", handler.ActivateAccount)
	}

	server := httptest.NewServer(router)
	return &serviceInstance{server: server, client: server.Client(), baseURL: server.URL}
}

func truncateCorporateTables(t *testing.T) {
	t.Helper()
	truncateTables(t)

	// Truncate corporate-specific tables if they exist
	corporateTables := []string{
		"corporate_rides",
		"corporate_invoices",
		"ride_policies",
		"cost_centers",
		"corporate_employees",
		"departments",
		"corporate_accounts",
	}

	for _, table := range corporateTables {
		_, _ = dbPool.Exec(context.Background(), fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
	}
}

// ============================================
// CORPORATE ACCOUNT CREATION TESTS
// ============================================

func (s *CorporateFlowTestSuite) TestCreateCorporateAccount_Success() {
	t := s.T()

	createReq := map[string]interface{}{
		"name":             "Acme Corporation",
		"legal_name":       "Acme Corporation Inc.",
		"primary_email":    "admin@acme.com",
		"billing_email":    "billing@acme.com",
		"billing_cycle":    "monthly",
		"payment_term_days": 30,
		"industry":         "Technology",
		"company_size":     "medium",
	}

	type accountResponse struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Status       string `json:"status"`
		BillingCycle string `json:"billing_cycle"`
	}

	createResp := doRequest[accountResponse](t, corporateServiceKey, http.MethodPost, "/api/v1/corporate/accounts", createReq, nil)
	require.True(t, createResp.Success)
	require.NotEmpty(t, createResp.Data.ID)
	require.Equal(t, "Acme Corporation", createResp.Data.Name)
	require.Equal(t, "pending", createResp.Data.Status)
	require.Equal(t, "monthly", createResp.Data.BillingCycle)
}

func (s *CorporateFlowTestSuite) TestCreateCorporateAccount_WithAddress() {
	t := s.T()

	createReq := map[string]interface{}{
		"name":          "Tech Startup LLC",
		"legal_name":    "Tech Startup LLC",
		"primary_email": "contact@techstartup.com",
		"billing_email": "finance@techstartup.com",
		"billing_cycle": "weekly",
		"address": map[string]interface{}{
			"line1":       "123 Innovation Way",
			"city":        "San Francisco",
			"state":       "CA",
			"postal_code": "94102",
			"country":     "USA",
		},
	}

	type accountResponse struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Status  string `json:"status"`
	}

	createResp := doRequest[accountResponse](t, corporateServiceKey, http.MethodPost, "/api/v1/corporate/accounts", createReq, nil)
	require.True(t, createResp.Success)
	require.NotEmpty(t, createResp.Data.ID)
}

func (s *CorporateFlowTestSuite) TestCreateCorporateAccount_MissingRequiredFields() {
	t := s.T()

	// Missing required fields
	createReq := map[string]interface{}{
		"name": "Incomplete Corp",
	}

	resp := doRawRequest(t, corporateServiceKey, http.MethodPost, "/api/v1/corporate/accounts", createReq, nil)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func (s *CorporateFlowTestSuite) TestActivateCorporateAccount_Success() {
	t := s.T()
	ctx := context.Background()

	// Create corporate account
	accountID := s.createCorporateAccount(t, "Pending Corp")

	// Verify it's pending
	var status string
	err := dbPool.QueryRow(ctx, `SELECT status FROM corporate_accounts WHERE id = $1`, accountID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "pending", status)

	// Admin activates the account
	activatePath := fmt.Sprintf("/api/v1/admin/corporate/accounts/%s/activate", accountID)
	activateResp := doRequest[map[string]interface{}](t, corporateServiceKey, http.MethodPost, activatePath, nil, authHeaders(s.admin.Token))
	require.True(t, activateResp.Success)

	// Verify it's now active
	err = dbPool.QueryRow(ctx, `SELECT status FROM corporate_accounts WHERE id = $1`, accountID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "active", status)
}

func (s *CorporateFlowTestSuite) TestListCorporateAccounts_AdminOnly() {
	t := s.T()

	// Create some accounts
	s.createCorporateAccount(t, "Corp One")
	s.createCorporateAccount(t, "Corp Two")

	// Admin can list
	type accountListItem struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	listResp := doRequest[[]accountListItem](t, corporateServiceKey, http.MethodGet, "/api/v1/admin/corporate/accounts", nil, authHeaders(s.admin.Token))
	require.True(t, listResp.Success)
	require.GreaterOrEqual(t, len(listResp.Data), 2)

	// Non-admin cannot list
	nonAdminResp := doRawRequest(t, corporateServiceKey, http.MethodGet, "/api/v1/admin/corporate/accounts", nil, authHeaders(s.rider.Token))
	defer nonAdminResp.Body.Close()
	require.Equal(t, http.StatusForbidden, nonAdminResp.StatusCode)
}

// ============================================
// EMPLOYEE MANAGEMENT TESTS
// ============================================

func (s *CorporateFlowTestSuite) TestInviteEmployee_Success() {
	t := s.T()

	// Create and activate corporate account
	accountID := s.createAndActivateAccount(t, "Employee Test Corp")

	// Invite employee
	inviteReq := map[string]interface{}{
		"email":         "employee@company.com",
		"first_name":    "John",
		"last_name":     "Doe",
		"role":          "user",
		"job_title":     "Software Engineer",
		"monthly_limit": 500.00,
		"per_ride_limit": 50.00,
	}

	invitePath := fmt.Sprintf("/api/v1/corporate/accounts/%s/employees", accountID)

	type inviteResponse struct {
		Employee map[string]interface{} `json:"employee"`
		Message  string                 `json:"message"`
	}

	inviteResp := doRequest[inviteResponse](t, corporateServiceKey, http.MethodPost, invitePath, inviteReq, authHeaders(s.admin.Token))
	require.True(t, inviteResp.Success)
	require.Contains(t, inviteResp.Data.Message, "successfully")
	require.NotNil(t, inviteResp.Data.Employee)
}

func (s *CorporateFlowTestSuite) TestInviteEmployee_AsManager() {
	t := s.T()
	ctx := context.Background()

	// Create and activate corporate account
	accountID := s.createAndActivateAccount(t, "Manager Test Corp")

	// First, invite a manager
	managerReq := map[string]interface{}{
		"email":      "manager@company.com",
		"first_name": "Jane",
		"last_name":  "Manager",
		"role":       "manager",
	}

	invitePath := fmt.Sprintf("/api/v1/corporate/accounts/%s/employees", accountID)
	managerResp := doRequest[map[string]interface{}](t, corporateServiceKey, http.MethodPost, invitePath, managerReq, authHeaders(s.admin.Token))
	require.True(t, managerResp.Success)

	// Verify manager was created
	var managerCount int
	err := dbPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM corporate_employees
		WHERE corporate_account_id = $1 AND role = 'manager'`,
		accountID).Scan(&managerCount)
	require.NoError(t, err)
	require.Equal(t, 1, managerCount)
}

func (s *CorporateFlowTestSuite) TestListEmployees_Success() {
	t := s.T()

	// Create and activate corporate account
	accountID := s.createAndActivateAccount(t, "List Employees Corp")

	// Invite multiple employees
	for i := 0; i < 3; i++ {
		inviteReq := map[string]interface{}{
			"email":      fmt.Sprintf("employee%d@company.com", i),
			"first_name": fmt.Sprintf("Employee%d", i),
			"last_name":  "Test",
			"role":       "user",
		}

		invitePath := fmt.Sprintf("/api/v1/corporate/accounts/%s/employees", accountID)
		inviteResp := doRequest[map[string]interface{}](t, corporateServiceKey, http.MethodPost, invitePath, inviteReq, authHeaders(s.admin.Token))
		require.True(t, inviteResp.Success)
	}

	// List employees
	listPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/employees", accountID)

	type employeeItem struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		Role      string `json:"role"`
	}

	listResp := doRequest[[]employeeItem](t, corporateServiceKey, http.MethodGet, listPath, nil, authHeaders(s.admin.Token))
	require.True(t, listResp.Success)
	require.GreaterOrEqual(t, len(listResp.Data), 3)
}

func (s *CorporateFlowTestSuite) TestInviteEmployee_DuplicateEmail() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "Duplicate Email Corp")

	inviteReq := map[string]interface{}{
		"email":      "duplicate@company.com",
		"first_name": "First",
		"last_name":  "Employee",
		"role":       "user",
	}

	invitePath := fmt.Sprintf("/api/v1/corporate/accounts/%s/employees", accountID)

	// First invite succeeds
	firstResp := doRequest[map[string]interface{}](t, corporateServiceKey, http.MethodPost, invitePath, inviteReq, authHeaders(s.admin.Token))
	require.True(t, firstResp.Success)

	// Second invite with same email should fail
	secondResp := doRawRequest(t, corporateServiceKey, http.MethodPost, invitePath, inviteReq, authHeaders(s.admin.Token))
	defer secondResp.Body.Close()
	require.Contains(t, []int{http.StatusConflict, http.StatusBadRequest}, secondResp.StatusCode)
}

// ============================================
// POLICY ENFORCEMENT TESTS
// ============================================

func (s *CorporateFlowTestSuite) TestCreatePolicy_BudgetLimit() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "Policy Test Corp")

	// Create budget limit policy
	policyReq := map[string]interface{}{
		"name":        "Monthly Budget Limit",
		"policy_type": "amount_limit",
		"rules": map[string]interface{}{
			"max_amount_per_month": 1000.00,
			"max_amount_per_ride":  100.00,
		},
	}

	policyPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/policies", accountID)

	type policyResponse struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		PolicyType string `json:"policy_type"`
		IsActive   bool   `json:"is_active"`
	}

	policyResp := doRequest[policyResponse](t, corporateServiceKey, http.MethodPost, policyPath, policyReq, authHeaders(s.admin.Token))
	require.True(t, policyResp.Success)
	require.NotEmpty(t, policyResp.Data.ID)
	require.Equal(t, "Monthly Budget Limit", policyResp.Data.Name)
	require.Equal(t, "amount_limit", policyResp.Data.PolicyType)
	require.True(t, policyResp.Data.IsActive)
}

func (s *CorporateFlowTestSuite) TestCreatePolicy_TimeRestriction() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "Time Policy Corp")

	// Create time restriction policy (work hours only)
	policyReq := map[string]interface{}{
		"name":        "Work Hours Only",
		"policy_type": "time_restriction",
		"rules": map[string]interface{}{
			"allowed_days":       []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
			"allowed_start_time": "06:00",
			"allowed_end_time":   "22:00",
		},
	}

	policyPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/policies", accountID)

	type policyResponse struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		PolicyType string `json:"policy_type"`
	}

	policyResp := doRequest[policyResponse](t, corporateServiceKey, http.MethodPost, policyPath, policyReq, authHeaders(s.admin.Token))
	require.True(t, policyResp.Success)
	require.Equal(t, "time_restriction", policyResp.Data.PolicyType)
}

func (s *CorporateFlowTestSuite) TestCreatePolicy_RideTypeRestriction() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "Ride Type Policy Corp")

	// Create policy to only allow economy rides
	policyReq := map[string]interface{}{
		"name":        "Economy Only",
		"policy_type": "ride_type_restriction",
		"rules": map[string]interface{}{
			"allowed_ride_types": []string{"economy"},
			"blocked_ride_types": []string{"premium", "xl"},
		},
	}

	policyPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/policies", accountID)

	type policyResponse struct {
		ID         string `json:"id"`
		PolicyType string `json:"policy_type"`
	}

	policyResp := doRequest[policyResponse](t, corporateServiceKey, http.MethodPost, policyPath, policyReq, authHeaders(s.admin.Token))
	require.True(t, policyResp.Success)
	require.Equal(t, "ride_type_restriction", policyResp.Data.PolicyType)
}

func (s *CorporateFlowTestSuite) TestCreatePolicy_ApprovalRequired() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "Approval Policy Corp")

	// Create policy requiring approval for high-value rides
	policyReq := map[string]interface{}{
		"name":        "Manager Approval Required",
		"policy_type": "approval_required",
		"rules": map[string]interface{}{
			"approval_threshold": 75.00,
		},
	}

	policyPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/policies", accountID)

	type policyResponse struct {
		ID         string `json:"id"`
		PolicyType string `json:"policy_type"`
	}

	policyResp := doRequest[policyResponse](t, corporateServiceKey, http.MethodPost, policyPath, policyReq, authHeaders(s.admin.Token))
	require.True(t, policyResp.Success)
	require.Equal(t, "approval_required", policyResp.Data.PolicyType)
}

func (s *CorporateFlowTestSuite) TestPolicy_DepartmentSpecific() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "Dept Policy Corp")

	// Create a department first
	deptReq := map[string]interface{}{
		"name":           "Engineering",
		"code":           "ENG",
		"budget_monthly": 5000.00,
	}

	deptPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/departments", accountID)

	type deptResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	deptResp := doRequest[deptResponse](t, corporateServiceKey, http.MethodPost, deptPath, deptReq, authHeaders(s.admin.Token))
	require.True(t, deptResp.Success)
	deptID := deptResp.Data.ID

	// Create department-specific policy
	policyReq := map[string]interface{}{
		"name":          "Engineering Budget",
		"policy_type":   "amount_limit",
		"department_id": deptID,
		"rules": map[string]interface{}{
			"max_amount_per_month": 2000.00,
		},
	}

	policyPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/policies", accountID)

	type policyResponse struct {
		ID           string  `json:"id"`
		DepartmentID *string `json:"department_id"`
	}

	policyResp := doRequest[policyResponse](t, corporateServiceKey, http.MethodPost, policyPath, policyReq, authHeaders(s.admin.Token))
	require.True(t, policyResp.Success)
	require.NotNil(t, policyResp.Data.DepartmentID)
}

// ============================================
// BILLING AND INVOICING TESTS
// ============================================

func (s *CorporateFlowTestSuite) TestGenerateInvoice_Success() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "Invoice Test Corp")

	// Generate invoice for a period
	now := time.Now()
	periodStart := now.AddDate(0, -1, 0)
	periodEnd := now

	invoiceReq := map[string]interface{}{
		"period_start": periodStart.Format("2006-01-02"),
		"period_end":   periodEnd.Format("2006-01-02"),
	}

	invoicePath := fmt.Sprintf("/api/v1/corporate/accounts/%s/invoices", accountID)

	type invoiceResponse struct {
		ID            string  `json:"id"`
		InvoiceNumber string  `json:"invoice_number"`
		Status        string  `json:"status"`
		TotalAmount   float64 `json:"total_amount"`
		RideCount     int     `json:"ride_count"`
	}

	invoiceResp := doRequest[invoiceResponse](t, corporateServiceKey, http.MethodPost, invoicePath, invoiceReq, authHeaders(s.admin.Token))
	require.True(t, invoiceResp.Success)
	require.NotEmpty(t, invoiceResp.Data.ID)
	require.NotEmpty(t, invoiceResp.Data.InvoiceNumber)
	require.Equal(t, "draft", invoiceResp.Data.Status)
}

func (s *CorporateFlowTestSuite) TestListInvoices_Success() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "List Invoices Corp")

	// Generate multiple invoices
	for i := 0; i < 3; i++ {
		now := time.Now()
		periodStart := now.AddDate(0, -(i+1), 0)
		periodEnd := now.AddDate(0, -i, 0)

		invoiceReq := map[string]interface{}{
			"period_start": periodStart.Format("2006-01-02"),
			"period_end":   periodEnd.Format("2006-01-02"),
		}

		invoicePath := fmt.Sprintf("/api/v1/corporate/accounts/%s/invoices", accountID)
		invoiceResp := doRequest[map[string]interface{}](t, corporateServiceKey, http.MethodPost, invoicePath, invoiceReq, authHeaders(s.admin.Token))
		require.True(t, invoiceResp.Success)
	}

	// List invoices
	listPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/invoices", accountID)

	type invoiceItem struct {
		ID            string `json:"id"`
		InvoiceNumber string `json:"invoice_number"`
		Status        string `json:"status"`
	}

	listResp := doRequest[[]invoiceItem](t, corporateServiceKey, http.MethodGet, listPath, nil, authHeaders(s.admin.Token))
	require.True(t, listResp.Success)
	require.GreaterOrEqual(t, len(listResp.Data), 3)
}

func (s *CorporateFlowTestSuite) TestGetDashboard_Success() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "Dashboard Corp")

	// Get dashboard
	dashboardPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/dashboard", accountID)

	type dashboardResponse struct {
		Account          map[string]interface{} `json:"account"`
		EmployeeCount    int                    `json:"employee_count"`
		ActiveEmployees  int                    `json:"active_employees"`
		DepartmentCount  int                    `json:"department_count"`
		PendingApprovals int                    `json:"pending_approvals"`
	}

	dashboardResp := doRequest[dashboardResponse](t, corporateServiceKey, http.MethodGet, dashboardPath, nil, authHeaders(s.admin.Token))
	require.True(t, dashboardResp.Success)
	require.NotNil(t, dashboardResp.Data.Account)
	require.GreaterOrEqual(t, dashboardResp.Data.EmployeeCount, 0)
}

func (s *CorporateFlowTestSuite) TestListCorporateRides_Success() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "Rides Corp")

	// List rides
	ridesPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/rides", accountID)

	type ridesResponse struct {
		Rides []map[string]interface{} `json:"rides"`
		Count int                      `json:"count"`
	}

	ridesResp := doRequest[ridesResponse](t, corporateServiceKey, http.MethodGet, ridesPath, nil, authHeaders(s.admin.Token))
	require.True(t, ridesResp.Success)
	require.GreaterOrEqual(t, ridesResp.Data.Count, 0)
}

// ============================================
// DEPARTMENT MANAGEMENT TESTS
// ============================================

func (s *CorporateFlowTestSuite) TestCreateDepartment_Success() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "Department Corp")

	deptReq := map[string]interface{}{
		"name":           "Sales",
		"code":           "SALES",
		"budget_monthly": 10000.00,
	}

	deptPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/departments", accountID)

	type deptResponse struct {
		ID            string   `json:"id"`
		Name          string   `json:"name"`
		Code          *string  `json:"code"`
		BudgetMonthly *float64 `json:"budget_monthly"`
		IsActive      bool     `json:"is_active"`
	}

	deptResp := doRequest[deptResponse](t, corporateServiceKey, http.MethodPost, deptPath, deptReq, authHeaders(s.admin.Token))
	require.True(t, deptResp.Success)
	require.NotEmpty(t, deptResp.Data.ID)
	require.Equal(t, "Sales", deptResp.Data.Name)
	require.True(t, deptResp.Data.IsActive)
}

func (s *CorporateFlowTestSuite) TestListDepartments_Success() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "Multi Dept Corp")

	// Create multiple departments
	departments := []string{"Engineering", "Sales", "Marketing", "HR"}
	for _, dept := range departments {
		deptReq := map[string]interface{}{
			"name": dept,
		}

		deptPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/departments", accountID)
		deptResp := doRequest[map[string]interface{}](t, corporateServiceKey, http.MethodPost, deptPath, deptReq, authHeaders(s.admin.Token))
		require.True(t, deptResp.Success)
	}

	// List departments
	listPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/departments", accountID)

	type deptItem struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	listResp := doRequest[[]deptItem](t, corporateServiceKey, http.MethodGet, listPath, nil, authHeaders(s.admin.Token))
	require.True(t, listResp.Success)
	require.GreaterOrEqual(t, len(listResp.Data), 4)
}

// ============================================
// CORPORATE USER PROFILE TESTS
// ============================================

func (s *CorporateFlowTestSuite) TestGetMyProfile_NotCorporateUser() {
	t := s.T()

	// Regular user should see they're not a corporate user
	type profileResponse struct {
		IsCorporateUser bool   `json:"is_corporate_user"`
		Message         string `json:"message,omitempty"`
	}

	profileResp := doRequest[profileResponse](t, corporateServiceKey, http.MethodGet, "/api/v1/corporate/me", nil, authHeaders(s.rider.Token))
	require.True(t, profileResp.Success)
	require.False(t, profileResp.Data.IsCorporateUser)
}

// ============================================
// PENDING APPROVALS TESTS
// ============================================

func (s *CorporateFlowTestSuite) TestGetPendingApprovals_Success() {
	t := s.T()

	accountID := s.createAndActivateAccount(t, "Approvals Corp")

	// Get pending approvals
	approvalsPath := fmt.Sprintf("/api/v1/corporate/accounts/%s/approvals", accountID)

	type approvalsResponse struct {
		PendingApprovals []map[string]interface{} `json:"pending_approvals"`
		Count            int                      `json:"count"`
	}

	approvalsResp := doRequest[approvalsResponse](t, corporateServiceKey, http.MethodGet, approvalsPath, nil, authHeaders(s.admin.Token))
	require.True(t, approvalsResp.Success)
	require.GreaterOrEqual(t, approvalsResp.Data.Count, 0)
}

// ============================================
// HELPER METHODS
// ============================================

func (s *CorporateFlowTestSuite) createCorporateAccount(t *testing.T, name string) uuid.UUID {
	createReq := map[string]interface{}{
		"name":             name,
		"legal_name":       name + " Inc.",
		"primary_email":    fmt.Sprintf("%s@example.com", name),
		"billing_email":    fmt.Sprintf("billing@%s.com", name),
		"billing_cycle":    "monthly",
		"payment_term_days": 30,
	}

	type accountResponse struct {
		ID string `json:"id"`
	}

	createResp := doRequest[accountResponse](t, corporateServiceKey, http.MethodPost, "/api/v1/corporate/accounts", createReq, nil)
	require.True(t, createResp.Success)

	accountID, err := uuid.Parse(createResp.Data.ID)
	require.NoError(t, err)

	return accountID
}

func (s *CorporateFlowTestSuite) createAndActivateAccount(t *testing.T, name string) uuid.UUID {
	accountID := s.createCorporateAccount(t, name)

	// Activate the account
	activatePath := fmt.Sprintf("/api/v1/admin/corporate/accounts/%s/activate", accountID)
	activateResp := doRequest[map[string]interface{}](t, corporateServiceKey, http.MethodPost, activatePath, nil, authHeaders(s.admin.Token))
	require.True(t, activateResp.Success)

	return accountID
}
