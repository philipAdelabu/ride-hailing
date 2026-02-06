package corporate

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/pkg/pagination"
)

// Handler handles HTTP requests for corporate accounts
type Handler struct {
	service *Service
}

// NewHandler creates a new corporate handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// ACCOUNT ENDPOINTS
// ========================================

// CreateAccount creates a new corporate account
// POST /api/v1/corporate/accounts
func (h *Handler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	account, err := h.service.CreateAccount(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create account")
		return
	}

	common.SuccessResponse(c, account)
}

// GetAccount gets a corporate account
// GET /api/v1/corporate/accounts/:id
func (h *Handler) GetAccount(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	account, err := h.service.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "account not found")
		return
	}

	common.SuccessResponse(c, account)
}

// GetDashboard gets the corporate dashboard
// GET /api/v1/corporate/accounts/:id/dashboard
func (h *Handler) GetDashboard(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	dashboard, err := h.service.GetDashboard(c.Request.Context(), accountID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get dashboard")
		return
	}

	common.SuccessResponse(c, dashboard)
}

// ListAccounts lists corporate accounts (admin)
// GET /api/v1/admin/corporate/accounts
func (h *Handler) ListAccounts(c *gin.Context) {
	params := pagination.ParseParams(c)

	var status *AccountStatus
	if s := c.Query("status"); s != "" {
		st := AccountStatus(s)
		status = &st
	}

	accounts, err := h.service.ListAccounts(c.Request.Context(), status, params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list accounts")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(len(accounts)))
	common.SuccessResponseWithMeta(c, accounts, meta)
}

// ActivateAccount activates a pending account
// POST /api/v1/admin/corporate/accounts/:id/activate
func (h *Handler) ActivateAccount(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	if err := h.service.ActivateAccount(c.Request.Context(), accountID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to activate account")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Account activated successfully",
	})
}

// ========================================
// DEPARTMENT ENDPOINTS
// ========================================

// CreateDepartment creates a new department
// POST /api/v1/corporate/accounts/:id/departments
func (h *Handler) CreateDepartment(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	var req struct {
		Name          string     `json:"name" binding:"required"`
		Code          *string    `json:"code,omitempty"`
		ManagerID     *uuid.UUID `json:"manager_id,omitempty"`
		BudgetMonthly *float64   `json:"budget_monthly,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	dept, err := h.service.CreateDepartment(c.Request.Context(), accountID, req.Name, req.Code, req.ManagerID, req.BudgetMonthly)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create department")
		return
	}

	common.SuccessResponse(c, dept)
}

// ListDepartments lists departments for an account
// GET /api/v1/corporate/accounts/:id/departments
func (h *Handler) ListDepartments(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	params := pagination.ParseParams(c)

	depts, err := h.service.ListDepartments(c.Request.Context(), accountID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list departments")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(len(depts)))
	common.SuccessResponseWithMeta(c, depts, meta)
}

// ========================================
// EMPLOYEE ENDPOINTS
// ========================================

// InviteEmployee invites an employee
// POST /api/v1/corporate/accounts/:id/employees
func (h *Handler) InviteEmployee(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	var req InviteEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	emp, err := h.service.InviteEmployee(c.Request.Context(), accountID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to invite employee")
		return
	}

	common.SuccessResponse(c, gin.H{
		"employee": emp,
		"message":  "Invitation sent successfully",
	})
}

// ListEmployees lists employees for an account
// GET /api/v1/corporate/accounts/:id/employees
func (h *Handler) ListEmployees(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	params := pagination.ParseParams(c)

	employees, err := h.service.ListEmployees(c.Request.Context(), accountID, params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list employees")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(len(employees)))
	common.SuccessResponseWithMeta(c, employees, meta)
}

// GetMyProfile gets the current user's corporate profile
// GET /api/v1/corporate/me
func (h *Handler) GetMyProfile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	emp, err := h.service.GetEmployeeByUserID(c.Request.Context(), userID)
	if err != nil {
		common.SuccessResponse(c, gin.H{
			"is_corporate_user": false,
			"message":           "Not a corporate user",
		})
		return
	}

	account, _ := h.service.GetAccount(c.Request.Context(), emp.CorporateAccountID)

	common.SuccessResponse(c, gin.H{
		"is_corporate_user": true,
		"employee":          emp,
		"account":           account,
	})
}

// ========================================
// RIDE ENDPOINTS
// ========================================

// ListRides lists corporate rides
// GET /api/v1/corporate/accounts/:id/rides
func (h *Handler) ListRides(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Parse date filters
	startDate := time.Now().AddDate(0, -1, 0) // Default: last month
	endDate := time.Now()

	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startDate = t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			endDate = t.Add(24 * time.Hour).Add(-time.Second) // End of day
		}
	}

	var employeeID *uuid.UUID
	if s := c.Query("employee_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			employeeID = &id
		}
	}

	rides, err := h.service.ListCorporateRides(c.Request.Context(), accountID, employeeID, startDate, endDate, limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list rides")
		return
	}

	common.SuccessResponse(c, gin.H{
		"rides": rides,
		"count": len(rides),
	})
}

// GetPendingApprovals gets rides pending approval
// GET /api/v1/corporate/accounts/:id/approvals
func (h *Handler) GetPendingApprovals(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	rides, err := h.service.GetPendingApprovals(c.Request.Context(), accountID, nil)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get pending approvals")
		return
	}

	common.SuccessResponse(c, gin.H{
		"pending_approvals": rides,
		"count":             len(rides),
	})
}

// ApproveRide approves or rejects a ride
// POST /api/v1/corporate/rides/:id/approve
func (h *Handler) ApproveRide(c *gin.Context) {
	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Approved bool `json:"approved" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ApproveRide(c.Request.Context(), rideID, userID, req.Approved); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to process approval")
		return
	}

	status := "approved"
	if !req.Approved {
		status = "rejected"
	}

	common.SuccessResponse(c, gin.H{
		"message": "Ride " + status + " successfully",
	})
}

// ========================================
// INVOICE ENDPOINTS
// ========================================

// ListInvoices lists invoices for an account
// GET /api/v1/corporate/accounts/:id/invoices
func (h *Handler) ListInvoices(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	params := pagination.ParseParams(c)

	invoices, err := h.service.ListInvoices(c.Request.Context(), accountID, params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list invoices")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(len(invoices)))
	common.SuccessResponseWithMeta(c, invoices, meta)
}

// GenerateInvoice generates an invoice for a period
// POST /api/v1/corporate/accounts/:id/invoices
func (h *Handler) GenerateInvoice(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	var req struct {
		PeriodStart string `json:"period_start" binding:"required"`
		PeriodEnd   string `json:"period_end" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	periodStart, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid period_start format")
		return
	}

	periodEnd, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid period_end format")
		return
	}

	invoice, err := h.service.GenerateInvoice(c.Request.Context(), accountID, periodStart, periodEnd)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to generate invoice")
		return
	}

	common.SuccessResponse(c, invoice)
}

// ========================================
// POLICY ENDPOINTS
// ========================================

// CreatePolicy creates a new policy
// POST /api/v1/corporate/accounts/:id/policies
func (h *Handler) CreatePolicy(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	var req struct {
		Name         string      `json:"name" binding:"required"`
		PolicyType   PolicyType  `json:"policy_type" binding:"required"`
		Rules        PolicyRules `json:"rules" binding:"required"`
		DepartmentID *uuid.UUID  `json:"department_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	policy, err := h.service.CreatePolicy(c.Request.Context(), accountID, req.Name, req.PolicyType, req.Rules, req.DepartmentID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create policy")
		return
	}

	common.SuccessResponse(c, policy)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers corporate routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Public corporate signup
	corporate := r.Group("/api/v1/corporate")
	{
		corporate.POST("/accounts", h.CreateAccount) // Anyone can request a corporate account
	}

	// Authenticated corporate routes
	corporateAuth := r.Group("/api/v1/corporate")
	corporateAuth.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		// User's corporate profile
		corporateAuth.GET("/me", h.GetMyProfile)

		// Account management (requires corporate admin role - simplified here)
		corporateAuth.GET("/accounts/:id", h.GetAccount)
		corporateAuth.GET("/accounts/:id/dashboard", h.GetDashboard)
		corporateAuth.GET("/accounts/:id/departments", h.ListDepartments)
		corporateAuth.POST("/accounts/:id/departments", h.CreateDepartment)
		corporateAuth.GET("/accounts/:id/employees", h.ListEmployees)
		corporateAuth.POST("/accounts/:id/employees", h.InviteEmployee)
		corporateAuth.GET("/accounts/:id/rides", h.ListRides)
		corporateAuth.GET("/accounts/:id/approvals", h.GetPendingApprovals)
		corporateAuth.GET("/accounts/:id/invoices", h.ListInvoices)
		corporateAuth.POST("/accounts/:id/invoices", h.GenerateInvoice)
		corporateAuth.POST("/accounts/:id/policies", h.CreatePolicy)

		// Ride approval
		corporateAuth.POST("/rides/:id/approve", h.ApproveRide)
	}

	// Admin routes
	admin := r.Group("/api/v1/admin/corporate")
	admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	{
		admin.GET("/accounts", h.ListAccounts)
		admin.POST("/accounts/:id/activate", h.ActivateAccount)
	}
}
